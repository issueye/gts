package stdlib

import (
	"sync"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

type eventListener struct {
	id   int64
	fn   *object.Function
	once bool
}

type eventEmitter struct {
	mu        sync.Mutex
	nextID    int64
	listeners map[string][]eventListener
	self      *object.Hash
}

func init() {
	module.RegisterNative("@std/events", func() (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initEventsModule(exports)
		return exports, nil
	})
}

func initEventsModule(exports *object.Hash) {
	setHashMember(exports, "EventEmitter", &object.Builtin{Name: "events.EventEmitter", Fn: eventsEventEmitter})
}

func eventsEventEmitter(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	emitter := &eventEmitter{listeners: make(map[string][]eventListener)}
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	emitter.self = obj
	extra := &object.GoObject{Value: emitter}

	setHashMember(obj, "__eventEmitter", extra)
	setHashMember(obj, "on", &object.Builtin{Name: "events.on", Fn: eventsOn, Extra: extra})
	setHashMember(obj, "once", &object.Builtin{Name: "events.once", Fn: eventsOnce, Extra: extra})
	setHashMember(obj, "off", &object.Builtin{Name: "events.off", Fn: eventsOff, Extra: extra})
	setHashMember(obj, "emit", &object.Builtin{Name: "events.emit", Fn: eventsEmit, Extra: extra})
	setHashMember(obj, "listeners", &object.Builtin{Name: "events.listeners", Fn: eventsListeners, Extra: extra})
	setHashMember(obj, "listenerCount", &object.Builtin{Name: "events.listenerCount", Fn: eventsListenerCount, Extra: extra})
	setHashMember(obj, "removeAllListeners", &object.Builtin{Name: "events.removeAllListeners", Fn: eventsRemoveAllListeners, Extra: extra})

	return obj
}

func eventsOn(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return eventsAddListener(env, pos, "events.on", false, args...)
}

func eventsOnce(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	return eventsAddListener(env, pos, "events.once", true, args...)
}

func eventsAddListener(env *object.Environment, pos ast.Position, name string, once bool, args ...object.Object) object.Object {
	emitter, errObj := currentEventEmitter(env, pos, name)
	if errObj != nil {
		return errObj
	}
	event, fn, errObj := eventAndFunction(pos, name, args)
	if errObj != nil {
		return errObj
	}

	emitter.mu.Lock()
	emitter.nextID++
	emitter.listeners[event] = append(emitter.listeners[event], eventListener{
		id: emitter.nextID, fn: fn, once: once,
	})
	emitter.mu.Unlock()

	return emitter.self
}

func eventsOff(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	emitter, errObj := currentEventEmitter(env, pos, "events.off")
	if errObj != nil {
		return errObj
	}
	event, fn, errObj := eventAndFunction(pos, "events.off", args)
	if errObj != nil {
		return errObj
	}

	emitter.mu.Lock()
	if entries, ok := emitter.listeners[event]; ok {
		next := entries[:0]
		removed := false
		for _, entry := range entries {
			if !removed && entry.fn == fn {
				removed = true
				continue
			}
			next = append(next, entry)
		}
		if len(next) == 0 {
			delete(emitter.listeners, event)
		} else {
			emitter.listeners[event] = next
		}
	}
	emitter.mu.Unlock()

	return emitter.self
}

func eventsEmit(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	emitter, errObj := currentEventEmitter(env, pos, "events.emit")
	if errObj != nil {
		return errObj
	}
	event, errObj := requiredString(pos, "events.emit", args, 0, "event")
	if errObj != nil {
		return errObj
	}

	emitter.mu.Lock()
	entries := append([]eventListener(nil), emitter.listeners[event]...)
	onceIDs := make(map[int64]bool)
	for _, entry := range entries {
		if entry.once {
			onceIDs[entry.id] = true
		}
	}
	if len(onceIDs) > 0 {
		emitter.removeListenersByIDLocked(event, onceIDs)
	}
	emitter.mu.Unlock()

	if len(entries) == 0 {
		return object.FALSE
	}

	callArgs := args[1:]
	for _, entry := range entries {
		result := callEventListener(entry.fn, emitter.self, callArgs)
		if object.IsRuntimeError(result) {
			return result
		}
	}
	return object.TRUE
}

func eventsListeners(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	emitter, errObj := currentEventEmitter(env, pos, "events.listeners")
	if errObj != nil {
		return errObj
	}
	event, errObj := requiredString(pos, "events.listeners", args, 0, "event")
	if errObj != nil {
		return errObj
	}

	emitter.mu.Lock()
	entries := emitter.listeners[event]
	listeners := make([]object.Object, len(entries))
	for i, entry := range entries {
		listeners[i] = entry.fn
	}
	emitter.mu.Unlock()

	return &object.Array{Elements: listeners}
}

func eventsListenerCount(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	emitter, errObj := currentEventEmitter(env, pos, "events.listenerCount")
	if errObj != nil {
		return errObj
	}
	event, errObj := requiredString(pos, "events.listenerCount", args, 0, "event")
	if errObj != nil {
		return errObj
	}

	emitter.mu.Lock()
	count := len(emitter.listeners[event])
	emitter.mu.Unlock()

	return &object.Number{Value: float64(count)}
}

func eventsRemoveAllListeners(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	emitter, errObj := currentEventEmitter(env, pos, "events.removeAllListeners")
	if errObj != nil {
		return errObj
	}

	emitter.mu.Lock()
	if len(args) == 0 || args[0] == object.UNDEFINED {
		emitter.listeners = make(map[string][]eventListener)
	} else {
		event, ok := args[0].(*object.String)
		if !ok {
			emitter.mu.Unlock()
			return object.NewError(pos, "events.removeAllListeners: event must be a string")
		}
		delete(emitter.listeners, event.Value)
	}
	emitter.mu.Unlock()

	return emitter.self
}

func currentEventEmitter(env *object.Environment, pos ast.Position, name string) (*eventEmitter, *object.Error) {
	extra, ok := env.Extra.(*object.GoObject)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid event emitter receiver", name)
	}
	emitter, ok := extra.Value.(*eventEmitter)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid event emitter receiver", name)
	}
	return emitter, nil
}

func eventAndFunction(pos ast.Position, name string, args []object.Object) (string, *object.Function, *object.Error) {
	event, errObj := requiredString(pos, name, args, 0, "event")
	if errObj != nil {
		return "", nil, errObj
	}
	if len(args) < 2 {
		return "", nil, object.NewError(pos, "%s requires listener", name)
	}
	fn, ok := args[1].(*object.Function)
	if !ok {
		return "", nil, object.NewError(pos, "%s: listener must be a function", name)
	}
	return event, fn, nil
}

func callEventListener(fn *object.Function, this object.Object, args []object.Object) object.Object {
	scope := fn.Env.NewScope()
	scope.Set("this", this)
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

func (e *eventEmitter) removeListenersByIDLocked(event string, ids map[int64]bool) {
	entries := e.listeners[event]
	next := entries[:0]
	for _, entry := range entries {
		if ids[entry.id] {
			continue
		}
		next = append(next, entry)
	}
	if len(next) == 0 {
		delete(e.listeners, event)
	} else {
		e.listeners[event] = next
	}
}
