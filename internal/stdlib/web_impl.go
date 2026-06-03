package stdlib

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

type webApp struct {
	mu     sync.RWMutex
	routes []webRoute
}

type webRoute struct {
	method   string
	path     string
	handlers []object.Object
}

type webContext struct {
	req       *http.Request
	writer    http.ResponseWriter
	body      string
	params    map[string]string
	mountPath string
	reqObj    object.Object
	resObj    object.Object
}

type webResponse struct {
	writer     http.ResponseWriter
	statusCode int
	wrote      bool
}

type webNativeMiddleware struct {
	name string
	fn   func(*webContext, object.Object) object.Object
}

func webCreateApp(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	app := &webApp{}
	return webAppObject(app)
}

func webJSON(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	middleware := &webNativeMiddleware{
		name: "web.json",
		fn: func(ctx *webContext, next object.Object) object.Object {
			reqHash, ok := ctx.reqObj.(*object.Hash)
			if !ok {
				return callWebNext(next)
			}
			if strings.TrimSpace(ctx.body) == "" {
				setHashMember(reqHash, "body", object.UNDEFINED)
				return callWebNext(next)
			}
			parsed, err := decodeWebJSON(ctx.body)
			if err != nil {
				ctx.writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
				ctx.writer.WriteHeader(http.StatusBadRequest)
				_, _ = ctx.writer.Write([]byte("invalid json body"))
				return object.UNDEFINED
			}
			setHashMember(reqHash, "body", parsed)
			return callWebNext(next)
		},
	}
	return &object.Builtin{Name: "web.json.middleware", Extra: &object.GoObject{Value: middleware}, Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
		return object.UNDEFINED
	}}
}

func webText(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	middleware := &webNativeMiddleware{
		name: "web.text",
		fn: func(ctx *webContext, next object.Object) object.Object {
			if reqHash, ok := ctx.reqObj.(*object.Hash); ok {
				setHashMember(reqHash, "body", &object.String{Value: ctx.body})
			}
			return callWebNext(next)
		},
	}
	return &object.Builtin{Name: "web.text.middleware", Extra: &object.GoObject{Value: middleware}, Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
		return object.UNDEFINED
	}}
}

func webStatic(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "web.static requires a root directory")
	}
	root, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "web.static: root must be a string")
	}
	rootDir := root.Value
	middleware := &webNativeMiddleware{
		name: "web.static",
		fn: func(ctx *webContext, next object.Object) object.Object {
			servedPath := webStaticRequestPath(ctx)
			if servedPath == "" {
				return callWebNext(next)
			}
			target := filepath.Clean(filepath.Join(rootDir, filepath.FromSlash(servedPath)))
			rootAbs, rootErr := filepath.Abs(rootDir)
			targetAbs, targetErr := filepath.Abs(target)
			rel, relErr := filepath.Rel(rootAbs, targetAbs)
			if rootErr != nil || targetErr != nil || relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				http.Error(ctx.writer, "forbidden", http.StatusForbidden)
				return object.UNDEFINED
			}
			stat, err := os.Stat(targetAbs)
			if err != nil {
				return callWebNext(next)
			}
			if stat.IsDir() {
				index := filepath.Join(targetAbs, "index.html")
				if _, err := os.Stat(index); err != nil {
					return callWebNext(next)
				}
				targetAbs = index
			}
			http.ServeFile(ctx.writer, ctx.req, targetAbs)
			return object.UNDEFINED
		},
	}
	return &object.Builtin{Name: "web.static.middleware", Extra: &object.GoObject{Value: middleware}, Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
		return object.UNDEFINED
	}}
}

func webAppObject(app *webApp) *object.Hash {
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(obj, "__webApp", &object.GoObject{Value: app})
	for _, method := range []string{"get", "post", "put", "patch", "delete", "all"} {
		m := method
		setHashMember(obj, m, &object.Builtin{Name: "web." + m, Extra: &object.GoObject{Value: app}, Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			return webRegisterRoute(env, pos, strings.ToUpper(m), args...)
		}})
	}
	setHashMember(obj, "use", &object.Builtin{Name: "web.use", Fn: webUse, Extra: &object.GoObject{Value: app}})
	setHashMember(obj, "listen", &object.Builtin{Name: "web.listen", Fn: webListen, Extra: &object.GoObject{Value: app}})
	return obj
}

func webUse(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	app, errObj := boundWebApp(pos, env, "web.use")
	if errObj != nil {
		return errObj
	}
	path := "/"
	fnIndex := 0
	if len(args) >= 2 {
		if s, ok := args[0].(*object.String); ok {
			path = s.Value
			fnIndex = 1
		}
	}
	if len(args) <= fnIndex {
		return object.NewError(pos, "web.use requires a handler")
	}
	handlers := args[fnIndex:]
	if !webHandlersValid(handlers) {
		return object.NewError(pos, "web.use: handler must be a function")
	}
	app.addRoute("USE", path, handlers)
	return object.UNDEFINED
}

func webRegisterRoute(env *object.Environment, pos ast.Position, method string, args ...object.Object) object.Object {
	app, errObj := boundWebApp(pos, env, "web."+strings.ToLower(method))
	if errObj != nil {
		return errObj
	}
	if len(args) < 2 {
		return object.NewError(pos, "web.%s requires path and handler", strings.ToLower(method))
	}
	path, ok := args[0].(*object.String)
	if !ok {
		return object.NewError(pos, "web.%s: path must be a string", strings.ToLower(method))
	}
	handlers := args[1:]
	if !webHandlersValid(handlers) {
		return object.NewError(pos, "web.%s: handler must be a function", strings.ToLower(method))
	}
	if method == "ALL" {
		method = "*"
	}
	app.addRoute(method, path.Value, handlers)
	return object.UNDEFINED
}

func webListen(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	app, errObj := boundWebApp(pos, env, "web.listen")
	if errObj != nil {
		return errObj
	}
	var port int
	if len(args) >= 1 {
		n, ok := args[0].(*object.Number)
		if !ok {
			return object.NewError(pos, "web.listen: port must be a number")
		}
		port = int(n.Value)
	}
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return object.NewError(pos, "web.listen: %v", err)
	}
	server := &http.Server{Handler: app}
	env.VM().Go(func() {
		_ = server.Serve(listener)
	})
	actualPort := port
	if tcpAddr, ok := listener.Addr().(*net.TCPAddr); ok {
		actualPort = tcpAddr.Port
	}
	result := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(result, "port", &object.Number{Value: float64(actualPort)})
	setHashMember(result, "address", &object.String{Value: listener.Addr().String()})
	setHashMember(result, "close", &object.Builtin{Name: "web.server.close", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
		if err := server.Close(); err != nil {
			return object.NewError(pos, "web.server.close: %v", err)
		}
		return object.UNDEFINED
	}})
	return result
}

func (app *webApp) addRoute(method, path string, handlers []object.Object) {
	if path == "" {
		path = "/"
	}
	copied := make([]object.Object, len(handlers))
	copy(copied, handlers)
	app.mu.Lock()
	app.routes = append(app.routes, webRoute{method: method, path: path, handlers: copied})
	app.mu.Unlock()
}

func (app *webApp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	_ = r.Body.Close()
	ctx := &webContext{req: r, writer: w, body: string(bodyBytes)}
	ctx.reqObj = buildWebRequestObject(ctx)
	ctx.resObj = newWebResponseObject(ctx.writer)
	routes := app.snapshotRoutes()
	app.runRoutes(routes, ctx, 0)
}

func (app *webApp) snapshotRoutes() []webRoute {
	app.mu.RLock()
	defer app.mu.RUnlock()
	routes := make([]webRoute, len(app.routes))
	copy(routes, app.routes)
	return routes
}

func (app *webApp) runRoutes(routes []webRoute, ctx *webContext, index int) {
	for i := index; i < len(routes); i++ {
		route := routes[i]
		params, ok := matchWebRoute(route, ctx.req)
		if !ok {
			continue
		}
		ctx.params = params
		ctx.mountPath = ""
		if route.method == "USE" {
			ctx.mountPath = route.path
		}
		updateWebRequestParams(ctx.reqObj, params)
		calledNext := false
		next := &object.Builtin{Name: "web.next", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			calledNext = true
			app.runRoutes(routes, ctx, i+1)
			return object.UNDEFINED
		}}
		callWebHandlers(route.handlers, ctx, next)
		if route.method != "USE" || !calledNext {
			return
		}
		return
	}
	http.NotFound(ctx.writer, ctx.req)
}

func matchWebRoute(route webRoute, r *http.Request) (map[string]string, bool) {
	if route.method != "USE" && route.method != "*" && route.method != r.Method {
		return nil, false
	}
	if route.method == "USE" {
		if route.path == "/" {
			return map[string]string{}, true
		}
		if r.URL.Path == route.path || strings.HasPrefix(r.URL.Path, strings.TrimRight(route.path, "/")+"/") {
			return map[string]string{}, true
		}
		return nil, false
	}
	return matchWebPath(route.path, r.URL.Path)
}

func matchWebPath(pattern, path string) (map[string]string, bool) {
	if pattern == path {
		return map[string]string{}, true
	}
	patternParts := splitWebPath(pattern)
	pathParts := splitWebPath(path)
	if len(patternParts) != len(pathParts) {
		return nil, false
	}
	params := make(map[string]string)
	for i, part := range patternParts {
		if strings.HasPrefix(part, ":") && len(part) > 1 {
			params[part[1:]] = pathParts[i]
			continue
		}
		if part != pathParts[i] {
			return nil, false
		}
	}
	return params, true
}

func splitWebPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "/")
}

func webStaticRequestPath(ctx *webContext) string {
	path := ctx.req.URL.Path
	if ctx.mountPath != "" && ctx.mountPath != "/" {
		prefix := strings.TrimRight(ctx.mountPath, "/")
		path = strings.TrimPrefix(path, prefix)
	}
	return strings.TrimPrefix(path, "/")
}

func webHandlersValid(handlers []object.Object) bool {
	if len(handlers) == 0 {
		return false
	}
	for _, handler := range handlers {
		switch handler.(type) {
		case *object.Function, *object.Builtin:
		default:
			return false
		}
	}
	return true
}

func callWebHandlers(handlers []object.Object, ctx *webContext, finalNext object.Object) object.Object {
	var run func(int) object.Object
	run = func(index int) object.Object {
		if index >= len(handlers) {
			return callWebNext(finalNext)
		}
		calledNext := false
		next := &object.Builtin{Name: "web.next", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
			calledNext = true
			return run(index + 1)
		}}
		result := callWebHandler(handlers[index], ctx, next)
		if calledNext {
			return result
		}
		return result
	}
	return run(0)
}

func callWebHandler(handler object.Object, ctx *webContext, next object.Object) object.Object {
	switch h := handler.(type) {
	case *object.Function:
		return callWebFunction(h, ctx.reqObj, ctx.resObj, next)
	case *object.Builtin:
		if native, ok := h.Extra.(*object.GoObject); ok {
			if middleware, ok := native.Value.(*webNativeMiddleware); ok {
				return middleware.fn(ctx, next)
			}
		}
		return h.Fn(&object.Environment{}, ast.Position{}, ctx.reqObj, ctx.resObj, next)
	default:
		return object.UNDEFINED
	}
}

func callWebFunction(fn *object.Function, req, res, next object.Object) object.Object {
	scope := fn.Env.NewScope()
	args := []object.Object{req, res, next}
	for i, p := range fn.Parameters {
		if i < len(args) {
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

func callWebNext(next object.Object) object.Object {
	if builtin, ok := next.(*object.Builtin); ok {
		return builtin.Fn(&object.Environment{}, ast.Position{})
	}
	return object.UNDEFINED
}

func buildWebRequestObject(ctx *webContext) object.Object {
	reqObj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(reqObj, "method", &object.String{Value: ctx.req.Method})
	setHashMember(reqObj, "url", &object.String{Value: ctx.req.URL.String()})
	setHashMember(reqObj, "path", &object.String{Value: ctx.req.URL.Path})
	setHashMember(reqObj, "body", &object.String{Value: ctx.body})
	setHashMember(reqObj, "rawBody", &object.String{Value: ctx.body})
	setHashMember(reqObj, "query", stringMapObject(firstQueryValues(ctx.req.URL.Query())))
	setHashMember(reqObj, "params", stringMapObject(ctx.params))
	setHashMember(reqObj, "headers", headerObject(ctx.req.Header))
	setHashMember(reqObj, "remoteAddr", &object.String{Value: ctx.req.RemoteAddr})
	return reqObj
}

func updateWebRequestParams(req object.Object, params map[string]string) {
	if hash, ok := req.(*object.Hash); ok {
		setHashMember(hash, "params", stringMapObject(params))
	}
}

func newWebResponseObject(w http.ResponseWriter) object.Object {
	resObj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	res := &webResponse{writer: w, statusCode: http.StatusOK}
	setHashMember(resObj, "status", &object.Builtin{Name: "web.response.status", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
		if len(args) < 1 {
			return object.NewError(pos, "response.status requires a status code")
		}
		n, ok := args[0].(*object.Number)
		if !ok {
			return object.NewError(pos, "response.status: code must be a number")
		}
		res.statusCode = int(n.Value)
		return resObj
	}})
	setHashMember(resObj, "setHeader", &object.Builtin{Name: "web.response.setHeader", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
		if len(args) < 2 {
			return object.NewError(pos, "response.setHeader requires key and value")
		}
		res.writer.Header().Set(args[0].Inspect(), args[1].Inspect())
		return resObj
	}})
	setHashMember(resObj, "send", &object.Builtin{Name: "web.response.send", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
		if len(args) < 1 {
			return object.UNDEFINED
		}
		if res.writer.Header().Get("Content-Type") == "" {
			res.writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		}
		res.write([]byte(args[0].Inspect()))
		return object.UNDEFINED
	}})
	setHashMember(resObj, "json", &object.Builtin{Name: "web.response.json", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
		if len(args) < 1 {
			return object.UNDEFINED
		}
		res.writer.Header().Set("Content-Type", "application/json")
		data, err := json.Marshal(toGoWebJSONValue(args[0]))
		if err != nil {
			return object.NewError(pos, "response.json: %v", err)
		}
		res.write(data)
		return object.UNDEFINED
	}})
	setHashMember(resObj, "redirect", &object.Builtin{Name: "web.response.redirect", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
		status := http.StatusFound
		targetIndex := 0
		if len(args) >= 2 {
			if n, ok := args[0].(*object.Number); ok {
				status = int(n.Value)
				targetIndex = 1
			}
		}
		if len(args) <= targetIndex {
			return object.NewError(pos, "response.redirect requires a URL")
		}
		res.writer.Header().Set("Location", args[targetIndex].Inspect())
		res.statusCode = status
		res.writeHeader()
		return object.UNDEFINED
	}})
	setHashMember(resObj, "end", &object.Builtin{Name: "web.response.end", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
		if len(args) > 0 {
			res.write([]byte(args[0].Inspect()))
		} else {
			res.writeHeader()
		}
		return object.UNDEFINED
	}})
	return resObj
}

func (res *webResponse) writeHeader() {
	if res.wrote {
		return
	}
	res.writer.WriteHeader(res.statusCode)
	res.wrote = true
}

func (res *webResponse) write(data []byte) {
	res.writeHeader()
	_, _ = res.writer.Write(data)
}

func boundWebApp(pos ast.Position, env *object.Environment, name string) (*webApp, *object.Error) {
	goObj, ok := env.Extra.(*object.GoObject)
	if !ok {
		return nil, object.NewError(pos, "%s: missing app receiver", name)
	}
	app, ok := goObj.Value.(*webApp)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid app receiver", name)
	}
	return app, nil
}

func stringMapObject(values map[string]string) *object.Hash {
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	for k, v := range values {
		setHashMember(obj, k, &object.String{Value: v})
	}
	return obj
}

func firstQueryValues(values map[string][]string) map[string]string {
	out := make(map[string]string)
	for k, vals := range values {
		if len(vals) > 0 {
			out[k] = vals[0]
		}
	}
	return out
}

func headerObject(headers http.Header) *object.Hash {
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	for k, vals := range headers {
		if len(vals) > 0 {
			setHashMember(obj, k, &object.String{Value: vals[0]})
		}
	}
	return obj
}

func decodeWebJSON(text string) (object.Object, error) {
	var raw interface{}
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}
	return goWebJSONToObject(raw), nil
}

func goWebJSONToObject(value interface{}) object.Object {
	switch v := value.(type) {
	case nil:
		return object.NULL
	case bool:
		return object.NativeBool(v)
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return object.NULL
		}
		return &object.Number{Value: f}
	case float64:
		return &object.Number{Value: v}
	case string:
		return &object.String{Value: v}
	case []interface{}:
		elements := make([]object.Object, len(v))
		for i, elem := range v {
			elements[i] = goWebJSONToObject(elem)
		}
		return &object.Array{Elements: elements}
	case map[string]interface{}:
		hash := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		for key, elem := range v {
			setHashMember(hash, key, goWebJSONToObject(elem))
		}
		return hash
	default:
		return object.UNDEFINED
	}
}

func toGoWebJSONValue(obj object.Object) interface{} {
	switch v := obj.(type) {
	case *object.Null, *object.Undefined:
		return nil
	case *object.Boolean:
		return v.Value
	case *object.Number:
		return v.Value
	case *object.String:
		return v.Value
	case *object.Array:
		out := make([]interface{}, len(v.Elements))
		for i, elem := range v.Elements {
			out[i] = toGoWebJSONValue(elem)
		}
		return out
	case *object.Hash:
		out := make(map[string]interface{})
		for _, pair := range v.Pairs {
			out[pair.Key.Inspect()] = toGoWebJSONValue(pair.Value)
		}
		return out
	case *object.Instance:
		out := make(map[string]interface{})
		for k, val := range v.Props {
			out[k] = toGoWebJSONValue(val)
		}
		return out
	default:
		return obj.Inspect()
	}
}
