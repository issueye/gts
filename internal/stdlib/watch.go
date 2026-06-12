package stdlib

import (
	"os"
	"time"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/evaluator"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

func init() {
	module.RegisterNative("@std/watch", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		setHashMember(exports, "file", &object.Builtin{Name: "watch.file", Fn: watchFile})
		return exports, nil
	})
}

func watchFile(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 2 {
		return object.NewError(pos, "watch.file requires path and callback")
	}
	path, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "watch.file expects string path")
	}
	fn, ok := args[1].(*object.Function)
	if !ok {
		return object.NewError(pos, "watch.file expects function callback")
	}

	interval := 1000 * time.Millisecond
	if len(args) > 2 {
		if opts, ok := args[2].(*object.Hash); ok {
			if pair, exists := opts.Pairs[object.HashKeyFor(&object.String{Value: "interval"})]; exists {
				if n, ok := pair.Value.(*object.Number); ok {
					interval = time.Duration(n.Value) * time.Millisecond
				}
			}
		}
	}

	go func() {
		var lastMod time.Time
		info, err := os.Stat(path.Value)
		if err == nil {
			lastMod = info.ModTime()
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			info, err := os.Stat(path.Value)
			if err != nil {
				continue
			}
			if info.ModTime().After(lastMod) {
				lastMod = info.ModTime()
				evaluator.Eval(fn.Body, fn.Env)
			}
		}
	}()

	return object.UNDEFINED
}
