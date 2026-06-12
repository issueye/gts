package stdlib

import (
	"sync"
	"time"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

type rateLimiter struct {
	mu       sync.Mutex
	tokens   float64
	capacity float64
	rate     float64
	lastTime time.Time
}

func init() {
	module.RegisterNative("@std/rate-limit", func(env *object.Environment) (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		setHashMember(exports, "create", &object.Builtin{Name: "rateLimit.create", Fn: rateLimitCreate})
		return exports, nil
	})
}

func rateLimitCreate(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	rate := 10.0
	capacity := 10.0

	if len(args) > 0 {
		if opts, ok := args[0].(*object.Hash); ok {
			if pair, exists := opts.Pairs[object.HashKeyFor(&object.String{Value: "rate"})]; exists {
				if n, ok := pair.Value.(*object.Number); ok {
					rate = n.Value
				}
			}
			if pair, exists := opts.Pairs[object.HashKeyFor(&object.String{Value: "capacity"})]; exists {
				if n, ok := pair.Value.(*object.Number); ok {
					capacity = n.Value
				}
			}
		}
	}

	limiter := &rateLimiter{
		tokens:   capacity,
		capacity: capacity,
		rate:     rate,
		lastTime: time.Now(),
	}

	inst := &object.Instance{Props: make(map[string]object.Object)}

	inst.Props["tryAcquire"] = &object.Builtin{
		Name: "rateLimit.tryAcquire",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			limiter.mu.Lock()
			defer limiter.mu.Unlock()

			now := time.Now()
			elapsed := now.Sub(limiter.lastTime).Seconds()
			limiter.tokens += elapsed * limiter.rate
			if limiter.tokens > limiter.capacity {
				limiter.tokens = limiter.capacity
			}
			limiter.lastTime = now

			if limiter.tokens >= 1 {
				limiter.tokens--
				return object.TRUE
			}
			return object.FALSE
		},
	}

	inst.Props["acquire"] = &object.Builtin{
		Name: "rateLimit.acquire",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			for {
				limiter.mu.Lock()
				now := time.Now()
				elapsed := now.Sub(limiter.lastTime).Seconds()
				limiter.tokens += elapsed * limiter.rate
				if limiter.tokens > limiter.capacity {
					limiter.tokens = limiter.capacity
				}
				limiter.lastTime = now

				if limiter.tokens >= 1 {
					limiter.tokens--
					limiter.mu.Unlock()
					return object.UNDEFINED
				}

				waitTime := (1 - limiter.tokens) / limiter.rate
				limiter.mu.Unlock()
				time.Sleep(time.Duration(waitTime * float64(time.Second)))
			}
		},
	}

	inst.Props["remaining"] = &object.Builtin{
		Name: "rateLimit.remaining",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			limiter.mu.Lock()
			defer limiter.mu.Unlock()
			return &object.Number{Value: limiter.tokens}
		},
	}

	return inst
}
