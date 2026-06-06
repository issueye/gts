package pluginhost

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/gtp"
	"github.com/issueye/goscript/internal/object"
	"github.com/issueye/goscript/internal/proj"
)

type Host struct {
	ProjectRoot string

	mu      sync.Mutex
	plugins map[string]*Plugin
}

type Plugin struct {
	Name   string
	Config proj.PluginConfig
	Ready  gtp.Frame

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	enc    *gtp.Encoder
	dec    *gtp.Decoder

	done      chan struct{}
	events    chan gtp.Frame
	pendingMu sync.Mutex
	pending   map[string]chan gtp.Frame
	nextCall  int64

	eventMu        sync.Mutex
	nextListener   int64
	eventListeners map[string][]*pluginEventListener
}

type pluginModuleBinding struct {
	plugin *Plugin
	module string
	self   *object.Hash
}

type pluginEventListener struct {
	id      int64
	module  string
	event   string
	fn      *object.Function
	once    bool
	tracked bool
	done    sync.Once
}

func New(projectRoot string) *Host {
	return &Host{
		ProjectRoot: projectRoot,
		plugins:     make(map[string]*Plugin),
	}
}

func (h *Host) StartConfigured(plugins map[string]proj.PluginConfig) error {
	names := make([]string, 0, len(plugins))
	for name := range plugins {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		cfg := plugins[name]
		if !cfg.AutoStart {
			continue
		}
		if err := h.Start(name, cfg); err != nil {
			h.Close()
			return err
		}
	}
	return nil
}

func (h *Host) Start(name string, cfg proj.PluginConfig) error {
	if cfg.Command == "" {
		return fmt.Errorf("plugin %s: command is required", name)
	}
	cwd := cfg.Cwd
	if cwd == "" {
		cwd = h.ProjectRoot
	}
	if cwd != "" && !filepath.IsAbs(cwd) {
		cwd = filepath.Join(h.ProjectRoot, cwd)
	}

	cmd := exec.Command(cfg.Command, cfg.Args...)
	cmd.Dir = cwd
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("plugin %s: stdin: %w", name, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("plugin %s: stdout: %w", name, err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("plugin %s: start: %w", name, err)
	}

	plugin := &Plugin{
		Name:           name,
		Config:         cfg,
		cmd:            cmd,
		stdin:          stdin,
		stdout:         stdout,
		enc:            gtp.NewEncoder(stdin),
		dec:            gtp.NewDecoder(stdout),
		done:           make(chan struct{}),
		events:         make(chan gtp.Frame, 128),
		pending:        make(map[string]chan gtp.Frame),
		eventListeners: make(map[string][]*pluginEventListener),
	}
	if err := plugin.handshake(); err != nil {
		plugin.Close()
		return err
	}
	go plugin.readLoop()

	h.mu.Lock()
	if existing := h.plugins[name]; existing != nil {
		existing.Close()
	}
	h.plugins[name] = plugin
	h.mu.Unlock()
	return nil
}

func (h *Host) Plugin(name string) (*Plugin, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	plugin, ok := h.plugins[name]
	return plugin, ok
}

func (h *Host) NativeModule(moduleName string, env *object.Environment) (object.Object, bool) {
	h.mu.Lock()
	plugins := make([]*Plugin, 0, len(h.plugins))
	for _, plugin := range h.plugins {
		plugins = append(plugins, plugin)
	}
	h.mu.Unlock()
	for _, plugin := range plugins {
		if plugin.HasModule(moduleName) {
			return plugin.NativeModule(moduleName, env), true
		}
	}
	return nil, false
}

func (h *Host) Close() {
	h.mu.Lock()
	plugins := make([]*Plugin, 0, len(h.plugins))
	for _, plugin := range h.plugins {
		plugins = append(plugins, plugin)
	}
	h.plugins = make(map[string]*Plugin)
	h.mu.Unlock()
	for _, plugin := range plugins {
		plugin.Close()
	}
}

func (p *Plugin) Events() <-chan gtp.Frame {
	return p.events
}

func (p *Plugin) HasModule(moduleName string) bool {
	modules, ok := p.Ready.Modules.(map[string]any)
	if ok {
		_, exists := modules[moduleName]
		return exists
	}
	if modules, ok := p.Ready.Modules.(map[string][]string); ok {
		_, exists := modules[moduleName]
		return exists
	}
	return false
}

func (p *Plugin) NativeModule(moduleName string, env *object.Environment) object.Object {
	methodNames := p.Methods(moduleName)
	exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	binding := &pluginModuleBinding{plugin: p, module: moduleName, self: exports}
	extra := &object.GoObject{Value: binding}
	for _, method := range methodNames {
		methodName := method
		exports.SetMember(&object.String{Value: methodName}, &object.Builtin{
			Name: moduleName + "." + methodName,
			Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
				values := make([]gtp.Value, len(args))
				for i, arg := range args {
					values[i] = gtp.FromObject(arg)
				}
				result, err := p.Call(moduleName, methodName, values)
				if err != nil {
					return object.NewError(pos, "HostError: %v", err)
				}
				return gtpValueToObject(env, result)
			},
		})
	}
	exports.SetMember(&object.String{Value: "on"}, &object.Builtin{Name: moduleName + ".on", Fn: pluginOn, Extra: extra})
	exports.SetMember(&object.String{Value: "once"}, &object.Builtin{Name: moduleName + ".once", Fn: pluginOnce, Extra: extra})
	exports.SetMember(&object.String{Value: "off"}, &object.Builtin{Name: moduleName + ".off", Fn: pluginOff, Extra: extra})
	exports.SetMember(&object.String{Value: "listenerCount"}, &object.Builtin{Name: moduleName + ".listenerCount", Fn: pluginListenerCount, Extra: extra})
	return exports
}

func (p *Plugin) Methods(moduleName string) []string {
	if modules, ok := p.Ready.Modules.(map[string][]string); ok {
		return append([]string{}, modules[moduleName]...)
	}
	if modules, ok := p.Ready.Modules.(map[string]any); ok {
		raw, ok := modules[moduleName].([]any)
		if !ok {
			return nil
		}
		methods := make([]string, 0, len(raw))
		for _, item := range raw {
			if method, ok := item.(string); ok {
				methods = append(methods, method)
			}
		}
		return methods
	}
	return nil
}

func (p *Plugin) Call(moduleName, method string, args []gtp.Value) (gtp.Value, error) {
	p.pendingMu.Lock()
	p.nextCall++
	id := fmt.Sprintf("%s-call-%d", p.Name, p.nextCall)
	ch := make(chan gtp.Frame, 1)
	p.pending[id] = ch
	p.pendingMu.Unlock()

	if err := p.enc.Encode(gtp.Frame{
		Version: gtp.Version,
		ID:      id,
		Type:    "call",
		Module:  moduleName,
		Method:  method,
		Args:    args,
	}); err != nil {
		p.pendingMu.Lock()
		delete(p.pending, id)
		p.pendingMu.Unlock()
		return gtp.Null(), err
	}

	response := <-ch
	if response.OK != nil && !*response.OK {
		if response.Error != nil {
			return gtp.Null(), fmt.Errorf("%s: %s", response.Error.Name, response.Error.Message)
		}
		return gtp.Null(), fmt.Errorf("plugin call failed")
	}
	if response.Result == nil {
		return gtp.Undefined(), nil
	}
	return *response.Result, nil
}

func (p *Plugin) Close() {
	select {
	case <-p.done:
		return
	default:
	}
	_ = p.stdin.Close()
	_ = p.stdout.Close()
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	_ = p.cmd.Wait()
	p.closeEventListeners()
	select {
	case <-p.done:
	default:
		close(p.done)
		close(p.events)
	}
}

func (p *Plugin) handshake() error {
	capabilities := p.Config.Capabilities
	if len(capabilities) == 0 {
		capabilities = []string{"call", "event"}
	}
	if err := p.enc.Encode(gtp.Frame{
		Version:      gtp.Version,
		ID:           "hello-" + p.Name,
		Type:         "hello",
		Runtime:      "gts",
		Protocol:     "gtp",
		Capabilities: capabilities,
		Modules:      p.Config.Modules,
	}); err != nil {
		return fmt.Errorf("plugin %s: hello: %w", p.Name, err)
	}
	ready, err := p.dec.Decode()
	if err != nil {
		return fmt.Errorf("plugin %s: ready: %w", p.Name, err)
	}
	if ready.Type == "error" {
		if ready.Error != nil {
			return fmt.Errorf("plugin %s: %s: %s", p.Name, ready.Error.Name, ready.Error.Message)
		}
		return fmt.Errorf("plugin %s: handshake failed", p.Name)
	}
	if ready.Type != "ready" {
		return fmt.Errorf("plugin %s: expected ready frame, got %s", p.Name, ready.Type)
	}
	p.Ready = ready
	return nil
}

func (p *Plugin) readLoop() {
	defer func() {
		p.closeEventListeners()
		select {
		case <-p.done:
		default:
			close(p.done)
			close(p.events)
		}
	}()
	for {
		frame, err := p.dec.Decode()
		if err != nil {
			return
		}
		if frame.Type == "event" {
			p.dispatchEvent(frame)
			select {
			case p.events <- frame:
			default:
			}
			continue
		}
		if frame.Type == "result" {
			p.pendingMu.Lock()
			ch := p.pending[frame.ID]
			delete(p.pending, frame.ID)
			p.pendingMu.Unlock()
			if ch != nil {
				ch <- frame
				close(ch)
			}
		}
	}
}

func pluginOn(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return pluginAddListener(env, pos, "plugin.on", false, args...)
}

func pluginOnce(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return pluginAddListener(env, pos, "plugin.once", true, args...)
}

func pluginAddListener(env *object.Environment, pos ast.Position, name string, once bool, args ...object.Object) object.Object {
	binding, errObj := pluginBindingFromExtra(env, pos, name)
	if errObj != nil {
		return errObj
	}
	event, fn, errObj := pluginEventAndFunction(pos, name, args)
	if errObj != nil {
		return errObj
	}

	listener := binding.plugin.addEventListener(binding.module, event, fn, once)
	_ = listener
	return binding.self
}

func pluginOff(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	binding, errObj := pluginBindingFromExtra(env, pos, "plugin.off")
	if errObj != nil {
		return errObj
	}
	event, fn, errObj := pluginEventAndFunction(pos, "plugin.off", args)
	if errObj != nil {
		return errObj
	}
	binding.plugin.removeEventListener(binding.module, event, fn)
	return binding.self
}

func pluginListenerCount(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	binding, errObj := pluginBindingFromExtra(env, pos, "plugin.listenerCount")
	if errObj != nil {
		return errObj
	}
	if len(args) < 1 {
		return object.NewError(pos, "plugin.listenerCount requires event")
	}
	event, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "plugin.listenerCount: event must be a string")
	}
	return &object.Number{Value: float64(binding.plugin.listenerCount(binding.module, event.Value))}
}

func pluginBindingFromExtra(env *object.Environment, pos ast.Position, name string) (*pluginModuleBinding, *object.Error) {
	extra, ok := env.Extra.(*object.GoObject)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid plugin receiver", name)
	}
	binding, ok := extra.Value.(*pluginModuleBinding)
	if !ok || binding.plugin == nil {
		return nil, object.NewError(pos, "%s: invalid plugin receiver", name)
	}
	return binding, nil
}

func pluginEventAndFunction(pos ast.Position, name string, args []object.Object) (string, *object.Function, *object.Error) {
	if len(args) < 1 {
		return "", nil, object.NewError(pos, "%s requires event", name)
	}
	event, ok := args[0].(*object.String)
	if !ok {
		return "", nil, object.NewError(pos, "%s: event must be a string", name)
	}
	if len(args) < 2 {
		return "", nil, object.NewError(pos, "%s requires listener", name)
	}
	fn, ok := args[1].(*object.Function)
	if !ok {
		return "", nil, object.NewError(pos, "%s: listener must be a function", name)
	}
	return event.Value, fn, nil
}

func (p *Plugin) addEventListener(moduleName, event string, fn *object.Function, once bool) *pluginEventListener {
	p.eventMu.Lock()
	defer p.eventMu.Unlock()
	p.nextListener++
	listener := &pluginEventListener{id: p.nextListener, module: moduleName, event: event, fn: fn, once: once, tracked: true}
	if listener.tracked {
		fn.Env.VM().AsyncAdd(1)
	}
	key := pluginEventKey(moduleName, event)
	p.eventListeners[key] = append(p.eventListeners[key], listener)
	return listener
}

func (p *Plugin) removeEventListener(moduleName, event string, fn *object.Function) bool {
	p.eventMu.Lock()
	defer p.eventMu.Unlock()
	key := pluginEventKey(moduleName, event)
	listeners := p.eventListeners[key]
	for i, listener := range listeners {
		if listener.fn != fn {
			continue
		}
		p.eventListeners[key] = append(listeners[:i], listeners[i+1:]...)
		if len(p.eventListeners[key]) == 0 {
			delete(p.eventListeners, key)
		}
		listener.finish()
		return true
	}
	return false
}

func (p *Plugin) listenerCount(moduleName, event string) int {
	p.eventMu.Lock()
	defer p.eventMu.Unlock()
	return len(p.eventListeners[pluginEventKey(moduleName, event)])
}

func (p *Plugin) dispatchEvent(frame gtp.Frame) {
	p.eventMu.Lock()
	key := pluginEventKey(frame.Module, frame.Event)
	listeners := append([]*pluginEventListener(nil), p.eventListeners[key]...)
	onceIDs := make(map[int64]bool)
	for _, listener := range listeners {
		if listener.once {
			onceIDs[listener.id] = true
		}
	}
	if len(onceIDs) > 0 {
		p.removeEventListenersByIDLocked(key, onceIDs)
	}
	p.eventMu.Unlock()

	for _, listener := range listeners {
		listener := listener
		eventObj := pluginEventObject(listener.fn.Env, frame)
		vm := listener.fn.Env.VM()
		vm.AsyncAdd(1)
		vm.Go(func() {
			defer vm.AsyncDone()
			if listener.once {
				defer listener.finish()
			}
			result := callPluginListener(listener.fn, eventObj)
			if object.IsRuntimeError(result) {
				fmt.Fprintln(os.Stderr, result.Inspect())
			}
		})
	}
}

func (p *Plugin) removeEventListenersByIDLocked(key string, ids map[int64]bool) {
	listeners := p.eventListeners[key]
	next := listeners[:0]
	for _, listener := range listeners {
		if ids[listener.id] {
			continue
		}
		next = append(next, listener)
	}
	if len(next) == 0 {
		delete(p.eventListeners, key)
	} else {
		p.eventListeners[key] = next
	}
}

func (p *Plugin) closeEventListeners() {
	p.eventMu.Lock()
	defer p.eventMu.Unlock()
	for _, listeners := range p.eventListeners {
		for _, listener := range listeners {
			listener.finish()
		}
	}
	p.eventListeners = make(map[string][]*pluginEventListener)
}

func (l *pluginEventListener) finish() {
	if !l.tracked {
		return
	}
	l.done.Do(func() {
		l.fn.Env.VM().AsyncDone()
	})
}

func pluginEventObject(env *object.Environment, frame gtp.Frame) *object.Hash {
	out := env.ObjectManager().NewHash()
	out.SetMember(&object.String{Value: "id"}, &object.String{Value: frame.ID})
	out.SetMember(&object.String{Value: "type"}, &object.String{Value: frame.Type})
	out.SetMember(&object.String{Value: "module"}, &object.String{Value: frame.Module})
	out.SetMember(&object.String{Value: "event"}, &object.String{Value: frame.Event})
	if frame.Data != nil {
		out.SetMember(&object.String{Value: "data"}, gtpValueToObject(env, *frame.Data))
	} else {
		out.SetMember(&object.String{Value: "data"}, object.UNDEFINED)
	}
	return out
}

func callPluginListener(fn *object.Function, event object.Object) object.Object {
	scope := fn.Env.NewScope()
	for i, p := range fn.Parameters {
		if i == 0 {
			scope.Set(p.Name, event)
		} else if p.Default != nil {
			scope.Set(p.Name, fn.Env.VM().EvalNode(p.Default, fn.Env))
		} else {
			scope.Set(p.Name, object.UNDEFINED)
		}
	}
	result := fn.Env.VM().EvalNode(fn.Body, scope)
	if rv, ok := result.(*object.ReturnValue); ok {
		return rv.Value
	}
	return result
}

func pluginEventKey(moduleName, event string) string {
	return moduleName + "\x00" + event
}

func gtpValueToObject(env *object.Environment, value gtp.Value) object.Object {
	switch value.Type {
	case "undefined":
		return object.UNDEFINED
	case "null":
		return object.NULL
	case "boolean":
		if b, ok := value.BoolValue(); ok {
			return object.NativeBool(b)
		}
		return object.FALSE
	case "number":
		if n, ok := value.NumberValue(); ok {
			return &object.Number{Value: n}
		}
		return object.NULL
	case "string", "bytes":
		if s, ok := value.StringValue(); ok {
			return &object.String{Value: s}
		}
		return &object.String{}
	case "array":
		items := make([]object.Object, len(value.Items))
		for i, item := range value.Items {
			items[i] = gtpValueToObject(env, item)
		}
		return env.ObjectManager().NewArray(items)
	case "object":
		hash := env.ObjectManager().NewHash()
		for key, item := range value.Fields {
			hash.SetMember(&object.String{Value: key}, gtpValueToObject(env, item))
		}
		return hash
	case "error":
		return object.NewNamedError(env.Pos, value.Name, value.Message)
	default:
		return object.UNDEFINED
	}
}
