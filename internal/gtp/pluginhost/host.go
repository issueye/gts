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
		Name:    name,
		Config:  cfg,
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		enc:     gtp.NewEncoder(stdin),
		dec:     gtp.NewDecoder(stdout),
		done:    make(chan struct{}),
		events:  make(chan gtp.Frame, 128),
		pending: make(map[string]chan gtp.Frame),
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
