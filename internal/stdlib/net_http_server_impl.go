package stdlib

import (
	"fmt"
	"io"
	"net/http"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func httpServerCreateServer(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	var handler *object.Function
	var port int64 = 0

	if len(args) >= 1 {
		if fn, ok := args[0].(*object.Function); ok {
			handler = fn
		}
	}
	if len(args) >= 2 {
		if n, ok := args[1].(*object.Number); ok {
			port = int64(n.Value)
		}
	}
	server := &http.Server{}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		r.Body.Close()
		reqObj := buildRequestObject(w, r, string(bodyBytes))
		if handler != nil {
			scope := handler.Env.NewScope()
			if len(handler.Parameters) > 0 {
				scope.Set(handler.Parameters[0].Name, reqObj)
			}
			if len(handler.Parameters) > 1 {
				resObj := newResponseObject(w)
				scope.Set(handler.Parameters[1].Name, resObj)
			}
			object.Spawn(func() {
				object.EvalFn(handler.Body, scope)
			})
		}
	})
	server.Handler = mux
	addr := fmt.Sprintf(":%d", port)
	server.Addr = addr
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		}
	}()
	result := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(result, "port", &object.Number{Value: float64(port)})
	setHashMember(result, "close", &object.Builtin{
		Name: "server.close",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if err := server.Close(); err != nil {
				return object.NewError(pos, "server.close: %v", err)
			}
			return object.UNDEFINED
		},
	})
	return result
}

func buildRequestObject(w http.ResponseWriter, r *http.Request, bodyStr string) object.Object {
	reqObj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(reqObj, "method", &object.String{Value: r.Method})
	setHashMember(reqObj, "url", &object.String{Value: r.URL.String()})
	setHashMember(reqObj, "path", &object.String{Value: r.URL.Path})
	setHashMember(reqObj, "body", &object.String{Value: bodyStr})
	queryObj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	for k, vals := range r.URL.Query() {
		if len(vals) > 0 {
			setHashMember(queryObj, k, &object.String{Value: vals[0]})
		}
	}
	setHashMember(reqObj, "query", queryObj)
	headers := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	for k, vals := range r.Header {
		if len(vals) > 0 {
			setHashMember(headers, k, &object.String{Value: vals[0]})
		}
	}
	setHashMember(reqObj, "headers", headers)
	setHashMember(reqObj, "remoteAddr", &object.String{Value: r.RemoteAddr})
	setHashMember(reqObj, "_raw", &object.GoObject{Value: r})
	setHashMember(reqObj, "_writer", &object.GoObject{Value: w})
	return reqObj
}

func newResponseObject(w http.ResponseWriter) object.Object {
	resObj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(resObj, "status", &object.Builtin{
		Name: "response.status",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 1 {
				return object.NewError(pos, "response.status requires a status code")
			}
			if n, ok := args[0].(*object.Number); ok {
				w.WriteHeader(int(n.Value))
			}
			return resObj
		},
	})
	setHashMember(resObj, "setHeader", &object.Builtin{
		Name: "response.setHeader",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 2 {
				return object.NewError(pos, "response.setHeader requires key and value")
			}
			w.Header().Set(args[0].Inspect(), args[1].Inspect())
			return object.UNDEFINED
		},
	})
	setHashMember(resObj, "send", &object.Builtin{
		Name: "response.send",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 1 {
				return object.UNDEFINED
			}
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(args[0].Inspect()))
			return object.UNDEFINED
		},
	})
	setHashMember(resObj, "json", &object.Builtin{
		Name: "response.json",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) < 1 {
				return object.UNDEFINED
			}
			w.Header().Set("Content-Type", "application/json")
			if hash, ok := args[0].(*object.Hash); ok {
				w.Write([]byte(toJSONString(hash)))
			} else {
				w.Write([]byte(args[0].Inspect()))
			}
			return object.UNDEFINED
		},
	})
	setHashMember(resObj, "end", &object.Builtin{
		Name: "response.end",
		Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			if len(args) > 0 {
				w.Write([]byte(args[0].Inspect()))
			}
			return object.UNDEFINED
		},
	})
	return resObj
}
