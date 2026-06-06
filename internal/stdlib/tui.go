package stdlib

import (
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

type tuiApp struct {
	vm      *object.VirtualMachine
	spec    *object.Hash
	state   object.Object
	session *terminalSession
	running bool
	stopped bool
}

type tuiRunOptions struct {
	raw              bool
	alternateScreen  bool
	hideCursor       bool
	mouse            bool
	bracketedPaste   bool
	diff             bool
	clip             bool
	full             bool
	tickMs           int
	resizeDebounceMs int
}

func init() {
	module.RegisterNative("@std/tui", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initTUIModule(exports)
		return exports, nil
	})
}

func initTUIModule(exports *object.Hash) {
	setHashMember(exports, "createApp", &object.Builtin{Name: "tui.createApp", Fn: tuiCreateApp})
	setHashMember(exports, "key", &object.Builtin{Name: "tui.key", Fn: tuiKey})
	setHashMember(exports, "text", &object.Builtin{Name: "tui.text", Fn: tuiText})
	setHashMember(exports, "resize", &object.Builtin{Name: "tui.resize", Fn: tuiResize})
	setHashMember(exports, "tick", &object.Builtin{Name: "tui.tick", Fn: tuiTick})
	setHashMember(exports, "box", &object.Builtin{Name: "tui.box", Fn: tuiBox})
	setHashMember(exports, "input", &object.Builtin{Name: "tui.input", Fn: tuiInput})
	setHashMember(exports, "row", &object.Builtin{Name: "tui.row", Fn: tuiRow})
	setHashMember(exports, "column", &object.Builtin{Name: "tui.column", Fn: tuiColumn})
	setHashMember(exports, "pad", &object.Builtin{Name: "tui.pad", Fn: tuiPad})
	setHashMember(exports, "statusBar", &object.Builtin{Name: "tui.statusBar", Fn: tuiStatusBar})
	setHashMember(exports, "style", &object.Builtin{Name: "tui.style", Fn: terminalStyle})
	setHashMember(exports, "stripAnsi", &object.Builtin{Name: "tui.stripAnsi", Fn: textStripAnsi})
	setHashMember(exports, "width", &object.Builtin{Name: "tui.width", Fn: textWidth})
	setHashMember(exports, "truncate", &object.Builtin{Name: "tui.truncate", Fn: textTruncateWidth})
}

func tuiCreateApp(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "tui.createApp requires spec")
	}
	spec, ok := args[0].(*object.Hash)
	if !ok {
		return object.NewError(pos, "tui.createApp: spec must be an object")
	}
	app := &tuiApp{vm: env.VM(), spec: spec, state: object.UNDEFINED}
	if initFn, ok := hashFunction(spec, "init"); ok {
		size := terminalSizeObjectFromCurrent()
		result := callTUIFunction(initFn, nil, []object.Object{size})
		if promise, ok := result.(*object.Promise); ok {
			result = promise.Wait()
		}
		if object.IsRuntimeError(result) {
			return result
		}
		app.state = result
	} else if value, ok := hashValue(spec, "state"); ok {
		app.state = value
	}
	return tuiAppObject(app)
}

func tuiAppObject(app *tuiApp) *object.Hash {
	extra := &object.GoObject{Value: app}
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(obj, "__tuiApp", extra)
	setHashMember(obj, "dispatch", &object.Builtin{Name: "tui.app.dispatch", Fn: tuiAppDispatch, Extra: extra})
	setHashMember(obj, "render", &object.Builtin{Name: "tui.app.render", Fn: tuiAppRender, Extra: extra})
	setHashMember(obj, "run", &object.Builtin{Name: "tui.app.run", Fn: tuiAppRun, Extra: extra})
	setHashMember(obj, "stop", &object.Builtin{Name: "tui.app.stop", Fn: tuiAppStop, Extra: extra})
	setHashMember(obj, "state", &object.Builtin{Name: "tui.app.state", Fn: tuiAppState, Extra: extra})
	return obj
}

func tuiAppDispatch(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	app, errObj := boundTUIApp(pos, env, "tui.app.dispatch")
	if errObj != nil {
		return errObj
	}
	msg := object.Object(object.UNDEFINED)
	if len(args) > 0 {
		msg = args[0]
	}
	if errObj := app.dispatch(msg); errObj != nil {
		return errObj
	}
	return app.state
}

func tuiAppRender(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	app, errObj := boundTUIApp(pos, env, "tui.app.render")
	if errObj != nil {
		return errObj
	}
	size := terminalSizeObjectFromCurrent()
	if len(args) > 0 && args[0] != object.UNDEFINED && args[0] != object.NULL {
		if h, ok := args[0].(*object.Hash); ok {
			size = h
		} else {
			return object.NewError(pos, "tui.app.render: size must be an object")
		}
	}
	frame, errObj := app.render(size)
	if errObj != nil {
		return errObj
	}
	return &object.String{Value: frame}
}

func tuiAppRun(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	app, errObj := boundTUIApp(pos, env, "tui.app.run")
	if errObj != nil {
		return errObj
	}
	opts, errObj := tuiParseRunOptions(pos, args)
	if errObj != nil {
		return errObj
	}
	return app.run(pos, opts)
}

func tuiAppStop(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	app, errObj := boundTUIApp(pos, env, "tui.app.stop")
	if errObj != nil {
		return errObj
	}
	app.stopped = true
	if app.session != nil {
		if err := app.session.stopSession(); err != nil {
			return object.NewError(pos, "tui.app.stop: %v", err)
		}
	}
	return object.UNDEFINED
}

func tuiAppState(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	app, errObj := boundTUIApp(pos, env, "tui.app.state")
	if errObj != nil {
		return errObj
	}
	return app.state
}

func (app *tuiApp) dispatch(msg object.Object) *object.Error {
	if updateFn, ok := hashFunction(app.spec, "update"); ok {
		result := callTUIFunction(updateFn, nil, []object.Object{app.state, msg})
		if promise, ok := result.(*object.Promise); ok {
			result = promise.Wait()
		}
		if object.IsRuntimeError(result) {
			if err, ok := result.(*object.Error); ok {
				return err
			}
			return object.NewError(ast.Position{}, "%s", result.Inspect())
		}
		if h, ok := result.(*object.Hash); ok {
			if next, ok := hashValue(h, "state"); ok {
				app.state = next
			} else {
				app.state = result
			}
			if quit, ok := tuiHashBool(h, "quit"); ok && quit {
				app.stopped = true
			}
			return nil
		}
		app.state = result
		return nil
	}
	if h, ok := msg.(*object.Hash); ok {
		if typ, ok := tuiHashString(h, "type"); ok && typ == "quit" {
			app.stopped = true
		}
	}
	return nil
}

func (app *tuiApp) render(size *object.Hash) (string, *object.Error) {
	if viewFn, ok := hashFunction(app.spec, "view"); ok {
		result := callTUIFunction(viewFn, nil, []object.Object{app.state, size})
		if promise, ok := result.(*object.Promise); ok {
			result = promise.Wait()
		}
		if object.IsRuntimeError(result) {
			if err, ok := result.(*object.Error); ok {
				return "", err
			}
			return "", object.NewError(ast.Position{}, "%s", result.Inspect())
		}
		return tuiFrameText(result), nil
	}
	return objectToText(app.state), nil
}

func (app *tuiApp) run(pos ast.Position, opts tuiRunOptions) object.Object {
	if app.running {
		return object.NewError(pos, "tui.app.run: app is already running")
	}
	app.running = true
	app.stopped = false
	defer func() { app.running = false }()

	session := &terminalSession{
		vm:               app.vm,
		restoreOnError:   true,
		restoreOnExit:    true,
		resizeDebounceMs: opts.resizeDebounceMs,
		events:           make(chan terminalEvent, 256),
		stop:             make(chan struct{}),
	}
	session.lastCols, session.lastRows = terminalGetSize()
	if opts.raw {
		raw, err := terminalMakeRaw()
		if err != nil {
			return object.NewError(pos, "tui.app.run: %v", err)
		}
		session.raw = raw
	}
	if opts.bracketedPaste {
		if _, err := os.Stdout.Write([]byte("\x1b[?2004h")); err != nil {
			_ = session.restore()
			return object.NewError(pos, "tui.app.run: %v", err)
		}
		session.bracketedPaste = true
	}
	if opts.mouse {
		if _, err := os.Stdout.Write([]byte("\x1b[?1000h\x1b[?1002h\x1b[?1006h")); err != nil {
			_ = session.restore()
			return object.NewError(pos, "tui.app.run: %v", err)
		}
		session.mouse = true
	}
	if opts.alternateScreen {
		if _, err := os.Stdout.Write([]byte("\x1b[?1049h")); err != nil {
			_ = session.restore()
			return object.NewError(pos, "tui.app.run: %v", err)
		}
		session.alternateScreen = true
	}
	if opts.hideCursor {
		if _, err := os.Stdout.Write([]byte("\x1b[?25l")); err != nil {
			_ = session.restore()
			return object.NewError(pos, "tui.app.run: %v", err)
		}
		session.cursorHidden = true
	}
	registerTerminalSession(session)
	app.session = session
	defer func() {
		_ = session.stopSession()
		app.session = nil
	}()

	go session.readInputLoop()
	go session.resizeLoop()
	if opts.tickMs > 0 {
		go tuiTickLoop(session, opts.tickMs)
	}

	if errObj := app.dispatch(tuiResizeMessage(session.lastCols, session.lastRows, true)); errObj != nil {
		_ = session.restore()
		return errObj
	}
	if errObj := app.renderToSession(pos, opts, true); errObj != nil {
		_ = session.restore()
		return errObj
	}

	for !app.stopped {
		select {
		case <-session.stop:
			app.stopped = true
		case event := <-session.events:
			msg := tuiTerminalEventMessage(event)
			if errObj := app.dispatch(msg); errObj != nil {
				_ = session.restore()
				return errObj
			}
			full := event.kind == "resize" || event.kind == "start"
			if errObj := app.renderToSession(pos, opts, full); errObj != nil {
				_ = session.restore()
				return errObj
			}
		}
	}
	return app.state
}

func (app *tuiApp) renderToSession(pos ast.Position, opts tuiRunOptions, full bool) *object.Error {
	if app.session == nil {
		return object.NewError(pos, "tui.app.run: missing terminal session")
	}
	cols, rows := terminalGetSize()
	size := terminalSizeObject(cols, rows)
	frame, errObj := app.render(size)
	if errObj != nil {
		return errObj
	}
	frameOpts := terminalFrameOptions{rows: rows, cols: cols, clip: opts.clip, diff: opts.diff, full: opts.full || full}
	if frameOpts.rows < 1 {
		frameOpts.rows = 1
	}
	if frameOpts.cols < 1 {
		frameOpts.cols = 1
	}
	app.session.mu.Lock()
	seq, next := terminalBuildFrameSequence(frame, frameOpts, app.session.previousFrame)
	if frameOpts.diff {
		app.session.previousFrame = next
		app.session.previousRows = frameOpts.rows
		app.session.previousCols = frameOpts.cols
	} else {
		app.session.previousFrame = nil
		app.session.previousRows = 0
		app.session.previousCols = 0
	}
	app.session.mu.Unlock()
	if _, err := os.Stdout.Write([]byte(seq)); err != nil {
		return object.NewError(pos, "tui.app.run: render: %v", err)
	}
	return nil
}

func tuiTickLoop(session *terminalSession, ms int) {
	ticker := time.NewTicker(time.Duration(ms) * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-session.stop:
			return
		case <-ticker.C:
			if !session.sendEvent(terminalEvent{kind: "tick"}) {
				return
			}
		}
	}
}

func tuiTerminalEventMessage(event terminalEvent) object.Object {
	switch event.kind {
	case "input":
		return tuiParseInputMessage(event.data)
	case "resize":
		return tuiResizeMessage(event.cols, event.rows, event.stable)
	case "tick":
		return tuiTickMessage()
	default:
		out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		setHashMember(out, "type", &object.String{Value: event.kind})
		return out
	}
}

func tuiKey(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	name, errObj := requiredString(pos, "tui.key", args, 0, "name")
	if errObj != nil {
		return errObj
	}
	out := tuiKeyMessage(name, "")
	if len(args) > 1 {
		setHashMember(out, "raw", &object.String{Value: objectToText(args[1])})
	}
	return out
}

func tuiText(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	value, errObj := requiredString(pos, "tui.text", args, 0, "value")
	if errObj != nil {
		return errObj
	}
	return tuiTextMessage(value, value)
}

func tuiResize(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	cols, errObj := requiredNumber(pos, "tui.resize", args, 0, "cols")
	if errObj != nil {
		return errObj
	}
	rows, errObj := requiredNumber(pos, "tui.resize", args, 1, "rows")
	if errObj != nil {
		return errObj
	}
	return tuiResizeMessage(int(cols), int(rows), true)
}

func tuiTick(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return tuiTickMessage()
}

func tuiParseInputMessage(data string) object.Object {
	if strings.HasPrefix(data, "\x1b[<") && strings.HasSuffix(data, "M") || strings.HasSuffix(data, "m") {
		if msg, ok := tuiParseSGRMouse(data); ok {
			return msg
		}
	}
	switch data {
	case "\x03":
		return tuiKeyMessage("ctrl+c", data)
	case "\x04":
		return tuiKeyMessage("ctrl+d", data)
	case "\r", "\n":
		return tuiKeyMessage("enter", data)
	case "\t":
		return tuiKeyMessage("tab", data)
	case "\x1b":
		return tuiKeyMessage("esc", data)
	case "\x7f", "\b":
		return tuiKeyMessage("backspace", data)
	case "\x1b[A":
		return tuiKeyMessage("up", data)
	case "\x1b[B":
		return tuiKeyMessage("down", data)
	case "\x1b[C":
		return tuiKeyMessage("right", data)
	case "\x1b[D":
		return tuiKeyMessage("left", data)
	case "\x1b[5~":
		return tuiKeyMessage("pageup", data)
	case "\x1b[6~":
		return tuiKeyMessage("pagedown", data)
	case "\x1b[H", "\x1b[1~":
		return tuiKeyMessage("home", data)
	case "\x1b[F", "\x1b[4~":
		return tuiKeyMessage("end", data)
	case "\x1b[3~":
		return tuiKeyMessage("delete", data)
	}
	if data != "" && utf8.ValidString(data) && !strings.ContainsRune(data, '\x1b') {
		return tuiTextMessage(data, data)
	}
	return tuiRawMessage(data)
}

func tuiKeyMessage(name, raw string) *object.Hash {
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(out, "type", &object.String{Value: "key"})
	setHashMember(out, "key", &object.String{Value: name})
	if raw != "" {
		setHashMember(out, "raw", &object.String{Value: raw})
	}
	return out
}

func tuiTextMessage(value, raw string) *object.Hash {
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(out, "type", &object.String{Value: "text"})
	setHashMember(out, "text", &object.String{Value: value})
	if raw != "" {
		setHashMember(out, "raw", &object.String{Value: raw})
	}
	return out
}

func tuiResizeMessage(cols, rows int, stable bool) *object.Hash {
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(out, "type", &object.String{Value: "resize"})
	setHashMember(out, "cols", &object.Number{Value: float64(cols)})
	setHashMember(out, "rows", &object.Number{Value: float64(rows)})
	setHashMember(out, "stable", object.NativeBool(stable))
	return out
}

func tuiTickMessage() *object.Hash {
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(out, "type", &object.String{Value: "tick"})
	setHashMember(out, "timeMs", &object.Number{Value: float64(time.Now().UnixMilli())})
	return out
}

func tuiRawMessage(raw string) *object.Hash {
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(out, "type", &object.String{Value: "raw"})
	setHashMember(out, "raw", &object.String{Value: raw})
	return out
}

func tuiParseSGRMouse(data string) (object.Object, bool) {
	release := strings.HasSuffix(data, "m")
	body := strings.TrimSuffix(strings.TrimSuffix(strings.TrimPrefix(data, "\x1b[<"), "M"), "m")
	parts := strings.Split(body, ";")
	if len(parts) != 3 {
		return nil, false
	}
	var nums [3]int
	for i, p := range parts {
		if p == "" {
			return nil, false
		}
		for _, r := range p {
			if r < '0' || r > '9' {
				return nil, false
			}
			nums[i] = nums[i]*10 + int(r-'0')
		}
	}
	out := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(out, "type", &object.String{Value: "mouse"})
	setHashMember(out, "button", &object.Number{Value: float64(nums[0])})
	setHashMember(out, "x", &object.Number{Value: float64(nums[1])})
	setHashMember(out, "y", &object.Number{Value: float64(nums[2])})
	if release {
		setHashMember(out, "action", &object.String{Value: "release"})
	} else {
		setHashMember(out, "action", &object.String{Value: "press"})
	}
	setHashMember(out, "raw", &object.String{Value: data})
	return out, true
}

func tuiBox(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	content := ""
	if len(args) > 0 {
		content = tuiFrameText(args[0])
	}
	opts := tuiBoxOptions{}
	if len(args) > 1 && args[1] != object.UNDEFINED && args[1] != object.NULL {
		hash, ok := args[1].(*object.Hash)
		if !ok {
			return object.NewError(pos, "tui.box: options must be an object")
		}
		if errObj := opts.parse(pos, hash); errObj != nil {
			return errObj
		}
	}
	return &object.String{Value: renderTUIBox(content, opts)}
}

type tuiBoxOptions struct {
	title   string
	width   int
	height  int
	padding int
	border  bool
}

func (o *tuiBoxOptions) parse(pos ast.Position, hash *object.Hash) *object.Error {
	o.border = true
	if title, ok := tuiHashString(hash, "title"); ok {
		o.title = title
	}
	if width, ok, errObj := tuiHashIntOption(pos, "tui.box", hash, "width"); errObj != nil {
		return errObj
	} else if ok {
		o.width = width
	}
	if height, ok, errObj := tuiHashIntOption(pos, "tui.box", hash, "height"); errObj != nil {
		return errObj
	} else if ok {
		o.height = height
	}
	if padding, ok, errObj := tuiHashIntOption(pos, "tui.box", hash, "padding"); errObj != nil {
		return errObj
	} else if ok {
		o.padding = padding
	}
	if border, ok := tuiHashBool(hash, "border"); ok {
		o.border = border
	}
	return nil
}

func renderTUIBox(content string, opts tuiBoxOptions) string {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		lines = nil
	}
	innerWidth := 0
	for _, line := range lines {
		if w := textVisibleWidth(line); w > innerWidth {
			innerWidth = w
		}
	}
	if opts.title != "" {
		if w := textVisibleWidth(opts.title) + 2; w > innerWidth {
			innerWidth = w
		}
	}
	if opts.width > 0 {
		innerWidth = opts.width
		if opts.border {
			innerWidth -= 2
		}
		innerWidth -= opts.padding * 2
	}
	if innerWidth < 0 {
		innerWidth = 0
	}
	pad := strings.Repeat(" ", maxInt(opts.padding, 0))
	body := make([]string, 0, len(lines)+opts.padding*2)
	blank := pad + strings.Repeat("", 0) + textPadToWidth("", innerWidth) + pad
	for i := 0; i < opts.padding; i++ {
		body = append(body, blank)
	}
	for _, line := range lines {
		body = append(body, pad+textPadToWidth(textTruncateToWidth(line, innerWidth), innerWidth)+pad)
	}
	for i := 0; i < opts.padding; i++ {
		body = append(body, blank)
	}
	if opts.height > 0 {
		target := opts.height
		if opts.border {
			target -= 2
		}
		for len(body) < target {
			body = append(body, blank)
		}
		if len(body) > target {
			body = body[:target]
		}
	}
	if !opts.border {
		return strings.Join(body, "\n")
	}
	width := innerWidth + opts.padding*2
	title := ""
	if opts.title != "" {
		title = " " + textTruncateToWidth(opts.title, maxInt(width-2, 0)) + " "
	}
	topFill := width - textVisibleWidth(title)
	if topFill < 0 {
		topFill = 0
	}
	top := "┌" + title + strings.Repeat("─", topFill) + "┐"
	bottom := "└" + strings.Repeat("─", width) + "┘"
	for i, line := range body {
		body[i] = "│" + textPadToWidth(line, width) + "│"
	}
	return strings.Join(append(append([]string{top}, body...), bottom), "\n")
}

type tuiInputOptions struct {
	title       string
	value       string
	cursor      int
	placeholder string
	prompt      string
	width       int
	focused     bool
	meta        string
}

func tuiInput(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "tui.input requires options")
	}
	hash, ok := args[0].(*object.Hash)
	if !ok {
		return object.NewError(pos, "tui.input: options must be an object")
	}
	opts := tuiInputOptions{
		title:   "Input",
		prompt:  "> ",
		width:   80,
		focused: true,
	}
	if errObj := opts.parse(pos, hash); errObj != nil {
		return errObj
	}
	return &object.String{Value: renderTUIInput(opts)}
}

func (o *tuiInputOptions) parse(pos ast.Position, hash *object.Hash) *object.Error {
	if title, ok := tuiHashString(hash, "title"); ok {
		o.title = title
	}
	if value, ok := tuiHashString(hash, "value"); ok {
		o.value = value
	}
	o.cursor = len(textVisibleChars(o.value))
	if cursor, ok, errObj := tuiHashIntOption(pos, "tui.input", hash, "cursor"); errObj != nil {
		return errObj
	} else if ok {
		o.cursor = cursor
	}
	if placeholder, ok := tuiHashString(hash, "placeholder"); ok {
		o.placeholder = placeholder
	}
	if prompt, ok := tuiHashString(hash, "prompt"); ok {
		o.prompt = prompt
	}
	if width, ok, errObj := tuiHashIntOption(pos, "tui.input", hash, "width"); errObj != nil {
		return errObj
	} else if ok {
		o.width = width
	}
	if focused, ok := tuiHashBool(hash, "focused"); ok {
		o.focused = focused
	}
	if meta, ok := tuiHashString(hash, "meta"); ok {
		o.meta = meta
	}
	if o.width < 1 {
		o.width = 1
	}
	if o.cursor < 0 {
		o.cursor = 0
	}
	if maxCursor := len(textVisibleChars(o.value)); o.cursor > maxCursor {
		o.cursor = maxCursor
	}
	return nil
}

func renderTUIInput(opts tuiInputOptions) string {
	width := opts.width
	inputWidth := width - textVisibleWidth(opts.prompt)
	if inputWidth < 1 {
		inputWidth = 1
	}
	lines := []string{
		terminalStyleString(textPadToWidth(opts.title, width), terminalStyleOptions{color: true, bold: true, fg: "accent"}),
		opts.prompt + renderTUIInputValue(opts, inputWidth),
	}
	if opts.meta != "" {
		lines = append(lines, terminalStyleString(textPadToWidth(opts.meta, width), terminalStyleOptions{color: true, dim: true, fg: "muted"}))
	}
	return strings.Join(lines, "\n")
}

func renderTUIInputValue(opts tuiInputOptions, width int) string {
	if opts.value == "" && opts.placeholder != "" {
		return terminalStyleString(textPadToWidth(textTruncateToWidth(opts.placeholder, width), width), terminalStyleOptions{color: true, dim: true, fg: "muted"})
	}
	if !opts.focused {
		return textPadToWidth(textTruncateToWidth(opts.value, width), width)
	}
	return cropTUIInputAroundCursor(opts.value, opts.cursor, width)
}

func cropTUIInputAroundCursor(value string, cursor, width int) string {
	if width < 1 {
		return ""
	}
	chars := textVisibleChars(value)
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(chars) {
		cursor = len(chars)
	}
	beforeBudget := (width - 1) * 62 / 100
	afterBudget := width - 1 - beforeBudget
	before := make([]string, 0, cursor)
	beforeWidth := 0
	for i := cursor - 1; i >= 0; i-- {
		ch := chars[i]
		next := textCharWidth(ch)
		if beforeWidth+next > beforeBudget {
			break
		}
		before = append([]string{ch}, before...)
		beforeWidth += next
	}
	after := make([]string, 0, len(chars)-cursor)
	afterWidth := 0
	for i := cursor; i < len(chars); i++ {
		ch := chars[i]
		next := textCharWidth(ch)
		if afterWidth+next > afterBudget {
			break
		}
		after = append(after, ch)
		afterWidth += next
	}
	row := strings.Join(before, "") + "\x1b[7m \x1b[0m" + strings.Join(after, "")
	return textPadToWidth(row, width)
}

func tuiRow(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	parts, errObj := tuiLayoutParts(pos, "tui.row", args)
	if errObj != nil {
		return errObj
	}
	return &object.String{Value: joinTUIHorizontal(parts)}
}

func tuiColumn(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	parts, errObj := tuiLayoutParts(pos, "tui.column", args)
	if errObj != nil {
		return errObj
	}
	return &object.String{Value: strings.Join(parts, "\n")}
}

func tuiPad(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "tui.pad requires content")
	}
	content := tuiFrameText(args[0])
	padding := 1
	if len(args) > 1 {
		n, ok := args[1].(*object.Number)
		if !ok {
			return object.NewError(pos, "tui.pad: padding must be a number")
		}
		padding = int(n.Value)
	}
	if padding < 0 {
		padding = 0
	}
	prefix := strings.Repeat(" ", padding)
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	for i, line := range lines {
		lines[i] = prefix + line + prefix
	}
	blankWidth := 0
	for _, line := range lines {
		if w := textVisibleWidth(line); w > blankWidth {
			blankWidth = w
		}
	}
	blank := strings.Repeat(" ", blankWidth)
	for i := 0; i < padding; i++ {
		lines = append([]string{blank}, lines...)
		lines = append(lines, blank)
	}
	return &object.String{Value: strings.Join(lines, "\n")}
}

func tuiStatusBar(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "tui.statusBar requires parts")
	}
	hash, ok := args[0].(*object.Hash)
	if !ok {
		return object.NewError(pos, "tui.statusBar: parts must be an object")
	}
	width := 80
	if len(args) > 1 {
		n, ok := args[1].(*object.Number)
		if !ok {
			return object.NewError(pos, "tui.statusBar: width must be a number")
		}
		width = int(n.Value)
	}
	if width < 1 {
		width = 1
	}
	left, _ := tuiHashString(hash, "left")
	center, _ := tuiHashString(hash, "center")
	right, _ := tuiHashString(hash, "right")
	left = textTruncateToWidth(left, width)
	rightBudget := width - textVisibleWidth(left) - 1
	if rightBudget < 0 {
		rightBudget = 0
	}
	right = textTruncateToWidth(right, rightBudget)
	line := left
	if center != "" && width >= textVisibleWidth(left)+textVisibleWidth(right)+textVisibleWidth(center)+2 {
		centerPos := (width - textVisibleWidth(center)) / 2
		line = textPadToWidth(line, centerPos) + center
	}
	line = textPadToWidth(line, width-textVisibleWidth(right)) + right
	return &object.String{Value: textTruncateToWidth(line, width)}
}

func tuiLayoutParts(pos ast.Position, name string, args []object.Object) ([]string, *object.Error) {
	if len(args) == 0 {
		return nil, nil
	}
	if arr, ok := args[0].(*object.Array); ok {
		parts := make([]string, len(arr.Elements))
		for i, item := range arr.Elements {
			parts[i] = tuiFrameText(item)
		}
		return parts, nil
	}
	parts := make([]string, len(args))
	for i, item := range args {
		parts[i] = tuiFrameText(item)
	}
	return parts, nil
}

func joinTUIHorizontal(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	blocks := make([][]string, len(parts))
	heights := 0
	widths := make([]int, len(parts))
	for i, part := range parts {
		lines := strings.Split(strings.ReplaceAll(part, "\r\n", "\n"), "\n")
		blocks[i] = lines
		if len(lines) > heights {
			heights = len(lines)
		}
		for _, line := range lines {
			if w := textVisibleWidth(line); w > widths[i] {
				widths[i] = w
			}
		}
	}
	out := make([]string, heights)
	for row := 0; row < heights; row++ {
		var b strings.Builder
		for col, lines := range blocks {
			line := ""
			if row < len(lines) {
				line = lines[row]
			}
			b.WriteString(textPadToWidth(line, widths[col]))
		}
		out[row] = b.String()
	}
	return strings.Join(out, "\n")
}

func tuiFrameText(value object.Object) string {
	if arr, ok := value.(*object.Array); ok {
		lines := make([]string, len(arr.Elements))
		for i, item := range arr.Elements {
			lines[i] = objectToText(item)
		}
		return strings.Join(lines, "\n")
	}
	return objectToText(value)
}

func tuiParseRunOptions(pos ast.Position, args []object.Object) (tuiRunOptions, *object.Error) {
	opts := tuiRunOptions{raw: true, alternateScreen: true, hideCursor: true, diff: true, clip: true, tickMs: 0, resizeDebounceMs: 50}
	if len(args) == 0 || args[0] == object.UNDEFINED || args[0] == object.NULL {
		return opts, nil
	}
	hash, ok := args[0].(*object.Hash)
	if !ok {
		return opts, object.NewError(pos, "tui.app.run: options must be an object")
	}
	if v, ok := tuiHashBool(hash, "raw"); ok {
		opts.raw = v
	}
	if v, ok := tuiHashBool(hash, "alternateScreen"); ok {
		opts.alternateScreen = v
	}
	if v, ok := tuiHashBool(hash, "hideCursor"); ok {
		opts.hideCursor = v
	}
	if v, ok := tuiHashBool(hash, "mouse"); ok {
		opts.mouse = v
	}
	if v, ok := tuiHashBool(hash, "bracketedPaste"); ok {
		opts.bracketedPaste = v
	}
	if v, ok := tuiHashBool(hash, "diff"); ok {
		opts.diff = v
	}
	if v, ok := tuiHashBool(hash, "clip"); ok {
		opts.clip = v
	}
	if v, ok := tuiHashBool(hash, "full"); ok {
		opts.full = v
	}
	if v, ok, errObj := tuiHashIntOption(pos, "tui.app.run", hash, "tickMs"); errObj != nil {
		return opts, errObj
	} else if ok {
		opts.tickMs = v
	}
	if v, ok, errObj := tuiHashIntOption(pos, "tui.app.run", hash, "resizeDebounceMs"); errObj != nil {
		return opts, errObj
	} else if ok {
		opts.resizeDebounceMs = v
	}
	if opts.tickMs < 0 {
		opts.tickMs = 0
	}
	if opts.resizeDebounceMs < 0 {
		opts.resizeDebounceMs = 0
	}
	return opts, nil
}

func terminalSizeObjectFromCurrent() *object.Hash {
	cols, rows := terminalGetSize()
	return terminalSizeObject(cols, rows)
}

func boundTUIApp(pos ast.Position, env *object.Environment, name string) (*tuiApp, *object.Error) {
	goObj, ok := env.Extra.(*object.GoObject)
	if !ok {
		return nil, object.NewError(pos, "%s: missing app receiver", name)
	}
	app, ok := goObj.Value.(*tuiApp)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid app receiver", name)
	}
	return app, nil
}

func hashFunction(hash *object.Hash, key string) (*object.Function, bool) {
	value, ok := hashValue(hash, key)
	if !ok {
		return nil, false
	}
	fn, ok := value.(*object.Function)
	return fn, ok
}

func tuiHashString(hash *object.Hash, key string) (string, bool) {
	value, ok := hashValue(hash, key)
	if !ok || value == object.UNDEFINED || value == object.NULL {
		return "", false
	}
	if s, ok := value.(*object.String); ok {
		return s.Value, true
	}
	return objectToText(value), true
}

func tuiHashBool(hash *object.Hash, key string) (bool, bool) {
	value, ok := hashValue(hash, key)
	if !ok || value == object.UNDEFINED || value == object.NULL {
		return false, false
	}
	if b, ok := value.(*object.Boolean); ok {
		return b.Value, true
	}
	return object.IsTruthy(value), true
}

func tuiHashIntOption(pos ast.Position, name string, hash *object.Hash, key string) (int, bool, *object.Error) {
	value, ok := hashValue(hash, key)
	if !ok || value == object.UNDEFINED || value == object.NULL {
		return 0, false, nil
	}
	n, ok := value.(*object.Number)
	if !ok {
		return 0, false, object.NewError(pos, "%s: %s must be a number", name, key)
	}
	return int(n.Value), true, nil
}

func callTUIFunction(fn *object.Function, this object.Object, args []object.Object) object.Object {
	scope := fn.Env.NewScope()
	if this != nil {
		scope.Set("this", this)
	}
	for i, p := range fn.Parameters {
		if i < len(args) {
			if p.Spread {
				rest := make([]object.Object, len(args)-i)
				copy(rest, args[i:])
				scope.Set(p.Name, fn.Env.ObjectManager().NewArray(rest))
				break
			}
			scope.Set(p.Name, args[i])
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

func textPadToWidth(value string, width int) string {
	if width <= 0 {
		return ""
	}
	out := value
	for textVisibleWidth(out) < width {
		out += " "
	}
	return out
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
