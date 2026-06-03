package stdlib

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

type signalWatcher struct {
	ch      chan os.Signal
	signals []os.Signal
}

func init() {
	module.RegisterNative("@std/signal", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initSignalModule(exports)
		return exports, nil
	})
}

func initSignalModule(exports *object.Hash) {
	setHashMember(exports, "supported", &object.Builtin{Name: "signal.supported", Fn: signalSupported})
	setHashMember(exports, "wait", &object.Builtin{Name: "signal.wait", Fn: signalWait})
	setHashMember(exports, "notify", &object.Builtin{Name: "signal.notify", Fn: signalNotify})
	setHashMember(exports, "send", &object.Builtin{Name: "signal.send", Fn: signalSend})

	for _, name := range sortedSignalNames() {
		setHashMember(exports, name, &object.String{Value: name})
	}
}

func signalSupported(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return strSliceToArray(sortedSignalNames())
}

func signalWait(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	signals, timeoutMs, errObj := signalWaitOptions(pos, "signal.wait", args)
	if errObj != nil {
		return errObj
	}
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, signals...)
	defer signal.Stop(ch)
	return waitForSignal(pos, "signal.wait", ch, timeoutMs)
}

func signalNotify(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	signals, _, errObj := signalWaitOptions(pos, "signal.notify", args)
	if errObj != nil {
		return errObj
	}
	watcher := &signalWatcher{ch: make(chan os.Signal, 1), signals: signals}
	signal.Notify(watcher.ch, signals...)
	return signalWatcherObject(watcher)
}

func signalSend(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	pid, errObj := requiredNumber(pos, "signal.send", args, 0, "pid")
	if errObj != nil {
		return errObj
	}
	sig := os.Interrupt
	if len(args) >= 2 && args[1] != object.UNDEFINED && args[1] != object.NULL {
		parsed, errObj := signalFromObject(pos, "signal.send", args[1])
		if errObj != nil {
			return errObj
		}
		sig = parsed
	}
	proc, err := os.FindProcess(int(pid))
	if err != nil {
		return object.NewError(pos, "signal.send: %v", err)
	}
	if err := proc.Signal(sig); err != nil {
		return object.NewError(pos, "signal.send: %v", err)
	}
	return object.UNDEFINED
}

func signalWatcherObject(watcher *signalWatcher) *object.Hash {
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(obj, "__signalWatcher", &object.GoObject{Value: watcher})
	setHashMember(obj, "wait", &object.Builtin{Name: "signal.watcher.wait", Fn: signalWatcherWait, Extra: &object.GoObject{Value: watcher}})
	setHashMember(obj, "stop", &object.Builtin{Name: "signal.watcher.stop", Fn: signalWatcherStop, Extra: &object.GoObject{Value: watcher}})
	return obj
}

func signalWatcherWait(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	watcher, errObj := boundSignalWatcher(pos, env, "signal.watcher.wait")
	if errObj != nil {
		return errObj
	}
	timeoutMs := -1
	if len(args) >= 1 && args[0] != object.UNDEFINED && args[0] != object.NULL {
		n, ok := args[0].(*object.Number)
		if !ok {
			return object.NewError(pos, "signal.watcher.wait: timeoutMs must be a number")
		}
		timeoutMs = int(n.Value)
	}
	return waitForSignal(pos, "signal.watcher.wait", watcher.ch, timeoutMs)
}

func signalWatcherStop(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	watcher, errObj := boundSignalWatcher(pos, env, "signal.watcher.stop")
	if errObj != nil {
		return errObj
	}
	signal.Stop(watcher.ch)
	return object.UNDEFINED
}

func boundSignalWatcher(pos ast.Position, env *object.Environment, name string) (*signalWatcher, *object.Error) {
	goObj, ok := env.Extra.(*object.GoObject)
	if !ok {
		return nil, object.NewError(pos, "%s: missing signal watcher receiver", name)
	}
	watcher, ok := goObj.Value.(*signalWatcher)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid signal watcher receiver", name)
	}
	return watcher, nil
}

func waitForSignal(pos ast.Position, name string, ch <-chan os.Signal, timeoutMs int) object.Object {
	if timeoutMs >= 0 {
		timer := time.NewTimer(time.Duration(timeoutMs) * time.Millisecond)
		defer timer.Stop()
		select {
		case sig := <-ch:
			return &object.String{Value: signalName(sig)}
		case <-timer.C:
			return object.NULL
		}
	}
	sig, ok := <-ch
	if !ok {
		return object.NewError(pos, "%s: signal channel closed", name)
	}
	return &object.String{Value: signalName(sig)}
}

func signalWaitOptions(pos ast.Position, name string, args []object.Object) ([]os.Signal, int, *object.Error) {
	timeoutMs := -1
	if len(args) == 0 || args[0] == object.UNDEFINED || args[0] == object.NULL {
		return []os.Signal{os.Interrupt, syscall.SIGTERM}, timeoutMs, nil
	}
	if hash, ok := args[0].(*object.Hash); ok {
		var signals []os.Signal
		if value, ok := hashValue(hash, "signals"); ok {
			parsed, errObj := signalListFromObject(pos, name, value)
			if errObj != nil {
				return nil, 0, errObj
			}
			signals = parsed
		}
		if value, ok := hashValue(hash, "timeoutMs"); ok {
			n, ok := value.(*object.Number)
			if !ok {
				return nil, 0, object.NewError(pos, "%s: timeoutMs must be a number", name)
			}
			timeoutMs = int(n.Value)
		}
		if len(signals) == 0 {
			signals = []os.Signal{os.Interrupt, syscall.SIGTERM}
		}
		return signals, timeoutMs, nil
	}
	signals, errObj := signalListFromObject(pos, name, args[0])
	if errObj != nil {
		return nil, 0, errObj
	}
	if len(args) >= 2 && args[1] != object.UNDEFINED && args[1] != object.NULL {
		n, ok := args[1].(*object.Number)
		if !ok {
			return nil, 0, object.NewError(pos, "%s: timeoutMs must be a number", name)
		}
		timeoutMs = int(n.Value)
	}
	return signals, timeoutMs, nil
}

func signalListFromObject(pos ast.Position, name string, value object.Object) ([]os.Signal, *object.Error) {
	if arr, ok := value.(*object.Array); ok {
		out := make([]os.Signal, 0, len(arr.Elements))
		for _, item := range arr.Elements {
			sig, errObj := signalFromObject(pos, name, item)
			if errObj != nil {
				return nil, errObj
			}
			out = append(out, sig)
		}
		return out, nil
	}
	sig, errObj := signalFromObject(pos, name, value)
	if errObj != nil {
		return nil, errObj
	}
	return []os.Signal{sig}, nil
}

func signalFromObject(pos ast.Position, name string, value object.Object) (os.Signal, *object.Error) {
	if n, ok := value.(*object.Number); ok {
		return syscall.Signal(int(n.Value)), nil
	}
	s, ok := value.(*object.String)
	if !ok {
		return nil, object.NewError(pos, "%s: signal must be a string or number", name)
	}
	sig, ok := signalByName(strings.ToUpper(s.Value))
	if !ok {
		return nil, object.NewError(pos, "%s: unsupported signal %q", name, s.Value)
	}
	return sig, nil
}

func signalByName(name string) (os.Signal, bool) {
	if !strings.HasPrefix(name, "SIG") {
		name = "SIG" + name
	}
	sig, ok := signalNameMap()[name]
	return sig, ok
}

func signalName(sig os.Signal) string {
	for name, candidate := range signalNameMap() {
		if candidate == sig {
			return name
		}
	}
	return sig.String()
}

func sortedSignalNames() []string {
	return []string{"SIGHUP", "SIGINT", "SIGQUIT", "SIGILL", "SIGTRAP", "SIGABRT", "SIGBUS", "SIGFPE", "SIGKILL", "SIGSEGV", "SIGPIPE", "SIGALRM", "SIGTERM"}
}

func signalNameMap() map[string]os.Signal {
	return map[string]os.Signal{
		"SIGHUP":  syscall.SIGHUP,
		"SIGINT":  syscall.SIGINT,
		"SIGQUIT": syscall.SIGQUIT,
		"SIGILL":  syscall.SIGILL,
		"SIGTRAP": syscall.SIGTRAP,
		"SIGABRT": syscall.SIGABRT,
		"SIGBUS":  syscall.SIGBUS,
		"SIGFPE":  syscall.SIGFPE,
		"SIGKILL": syscall.SIGKILL,
		"SIGSEGV": syscall.SIGSEGV,
		"SIGPIPE": syscall.SIGPIPE,
		"SIGALRM": syscall.SIGALRM,
		"SIGTERM": syscall.SIGTERM,
	}
}
