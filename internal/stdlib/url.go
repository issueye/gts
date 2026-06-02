package stdlib

import (
	"net/url"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/module"
	"github.com/issueye/goscript/internal/object"
)

type urlSearchParamsState struct {
	values url.Values
	parent *object.Hash
}

func init() {
	module.RegisterNative("@std/url", func() (object.Object, error) {
		exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		initURLModule(exports)
		return exports, nil
	})
}

func initURLModule(exports *object.Hash) {
	setHashMember(exports, "parse", &object.Builtin{Name: "url.parse", Fn: urlParse})
	setHashMember(exports, "format", &object.Builtin{Name: "url.format", Fn: urlFormat})
	setHashMember(exports, "resolve", &object.Builtin{Name: "url.resolve", Fn: urlResolve})
	setHashMember(exports, "pathToFileURL", &object.Builtin{Name: "url.pathToFileURL", Fn: urlPathToFileURL})
	setHashMember(exports, "fileURLToPath", &object.Builtin{Name: "url.fileURLToPath", Fn: urlFileURLToPath})
	setHashMember(exports, "URL", callableURLBuiltin("URL", urlConstructor))
	setHashMember(exports, "URLSearchParams", callableURLBuiltin("URLSearchParams", urlSearchParamsConstructor))
}

func callableURLBuiltin(name string, fn object.BuiltinFunc) *object.Hash {
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(obj, "__call", &object.Builtin{Name: name, Fn: fn})
	return obj
}

func urlParse(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	input, errObj := requiredString(pos, "url.parse", args, 0, "url")
	if errObj != nil {
		return errObj
	}
	u, err := url.Parse(input)
	if err != nil {
		return object.NewError(pos, "url.parse: %v", err)
	}
	return newURLObject(u)
}

func urlFormat(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) < 1 {
		return object.NewError(pos, "url.format requires a URL object")
	}
	u, errObj := urlFromObject(pos, "url.format", args[0])
	if errObj != nil {
		return errObj
	}
	return &object.String{Value: u.String()}
}

func urlResolve(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	base, errObj := requiredString(pos, "url.resolve", args, 0, "base")
	if errObj != nil {
		return errObj
	}
	ref, errObj := requiredString(pos, "url.resolve", args, 1, "ref")
	if errObj != nil {
		return errObj
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return object.NewError(pos, "url.resolve: invalid base URL: %v", err)
	}
	refURL, err := url.Parse(ref)
	if err != nil {
		return object.NewError(pos, "url.resolve: invalid ref URL: %v", err)
	}
	return &object.String{Value: baseURL.ResolveReference(refURL).String()}
}

func urlPathToFileURL(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	path, errObj := requiredString(pos, "url.pathToFileURL", args, 0, "path")
	if errObj != nil {
		return errObj
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return object.NewError(pos, "url.pathToFileURL: %v", err)
	}
	slashPath := filepath.ToSlash(abs)
	if runtime.GOOS == "windows" && !strings.HasPrefix(slashPath, "/") {
		slashPath = "/" + slashPath
	}
	u := &url.URL{Scheme: "file", Path: slashPath}
	return &object.String{Value: u.String()}
}

func urlFileURLToPath(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	input, errObj := requiredString(pos, "url.fileURLToPath", args, 0, "url")
	if errObj != nil {
		return errObj
	}
	u, err := url.Parse(input)
	if err != nil {
		return object.NewError(pos, "url.fileURLToPath: %v", err)
	}
	if u.Scheme != "file" {
		return object.NewError(pos, "url.fileURLToPath: URL must use file: protocol")
	}
	if u.Host != "" && u.Host != "localhost" {
		return object.NewError(pos, "url.fileURLToPath: file URL host is not supported")
	}
	path, err := url.PathUnescape(u.Path)
	if err != nil {
		return object.NewError(pos, "url.fileURLToPath: %v", err)
	}
	if runtime.GOOS == "windows" && strings.HasPrefix(path, "/") && len(path) >= 3 && path[2] == ':' {
		path = path[1:]
	}
	return &object.String{Value: filepath.FromSlash(path)}
}

func urlConstructor(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	input, errObj := requiredString(pos, "URL", args, 0, "input")
	if errObj != nil {
		return errObj
	}
	var u *url.URL
	var err error
	if len(args) > 1 && args[1] != object.UNDEFINED && args[1] != object.NULL {
		baseObj, ok := args[1].(*object.String)
		if !ok {
			return object.NewError(pos, "URL: base must be a string")
		}
		base, parseErr := url.Parse(baseObj.Value)
		if parseErr != nil {
			return object.NewError(pos, "URL: invalid base URL: %v", parseErr)
		}
		ref, parseErr := url.Parse(input)
		if parseErr != nil {
			return object.NewError(pos, "URL: invalid input URL: %v", parseErr)
		}
		u = base.ResolveReference(ref)
	} else {
		u, err = url.Parse(input)
		if err != nil {
			return object.NewError(pos, "URL: %v", err)
		}
	}
	return newURLObject(u)
}

func newURLObject(u *url.URL) *object.Hash {
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	populateURLObject(obj, u)
	return obj
}

func populateURLObject(obj *object.Hash, u *url.URL) {
	href := u.String()
	protocol := ""
	if u.Scheme != "" {
		protocol = u.Scheme + ":"
	}
	search := ""
	if u.RawQuery != "" {
		search = "?" + u.RawQuery
	}
	hash := ""
	if u.Fragment != "" {
		hash = "#" + u.Fragment
	}
	origin := "null"
	if u.Scheme != "" && u.Host != "" {
		origin = u.Scheme + "://" + u.Host
	}

	setHashMember(obj, "href", &object.String{Value: href})
	setHashMember(obj, "protocol", &object.String{Value: protocol})
	setHashMember(obj, "host", &object.String{Value: u.Host})
	setHashMember(obj, "hostname", &object.String{Value: u.Hostname()})
	setHashMember(obj, "port", &object.String{Value: u.Port()})
	setHashMember(obj, "pathname", &object.String{Value: u.EscapedPath()})
	setHashMember(obj, "search", &object.String{Value: search})
	setHashMember(obj, "hash", &object.String{Value: hash})
	setHashMember(obj, "origin", &object.String{Value: origin})
	setHashMember(obj, "searchParams", newURLSearchParamsObject(u.Query(), obj))
	setHashMember(obj, "toString", &object.Builtin{Name: "URL.toString", Fn: urlObjectToString, Extra: obj})
	setHashMember(obj, "toJSON", &object.Builtin{Name: "URL.toJSON", Fn: urlObjectToString, Extra: obj})
}

func urlObjectToString(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	obj, ok := env.Extra.(*object.Hash)
	if !ok {
		return object.NewError(pos, "URL.toString: missing URL receiver")
	}
	if href, ok := hashString(obj, "href"); ok {
		return &object.String{Value: href}
	}
	return object.NewError(pos, "URL.toString: invalid URL receiver")
}

func urlFromObject(pos ast.Position, name string, obj object.Object) (*url.URL, *object.Error) {
	switch v := obj.(type) {
	case *object.String:
		u, err := url.Parse(v.Value)
		if err != nil {
			return nil, object.NewError(pos, "%s: %v", name, err)
		}
		return u, nil
	case *object.Hash:
		if href, ok := hashString(v, "href"); ok && href != "" {
			u, err := url.Parse(href)
			if err != nil {
				return nil, object.NewError(pos, "%s: %v", name, err)
			}
			return u, nil
		}
		u := &url.URL{}
		if protocol, ok := hashString(v, "protocol"); ok {
			u.Scheme = strings.TrimSuffix(protocol, ":")
		}
		host, _ := hashString(v, "host")
		hostname, _ := hashString(v, "hostname")
		port, _ := hashString(v, "port")
		if host != "" {
			u.Host = host
		} else if hostname != "" {
			u.Host = hostname
			if port != "" {
				u.Host += ":" + port
			}
		}
		if pathname, ok := hashString(v, "pathname"); ok {
			u.Path = pathname
		}
		if search, ok := hashString(v, "search"); ok {
			u.RawQuery = strings.TrimPrefix(search, "?")
		}
		if fragment, ok := hashString(v, "hash"); ok {
			u.Fragment = strings.TrimPrefix(fragment, "#")
		}
		return u, nil
	default:
		return nil, object.NewError(pos, "%s: URL object must be an object or string", name)
	}
}

func urlSearchParamsConstructor(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	if len(args) == 0 || args[0] == object.UNDEFINED || args[0] == object.NULL {
		return newURLSearchParamsObject(url.Values{}, nil)
	}
	values, errObj := urlSearchParamsValues(pos, "URLSearchParams", args[0])
	if errObj != nil {
		return errObj
	}
	return newURLSearchParamsObject(values, nil)
}

func urlSearchParamsValues(pos ast.Position, name string, init object.Object) (url.Values, *object.Error) {
	values := url.Values{}
	switch v := init.(type) {
	case *object.String:
		parsed, err := url.ParseQuery(strings.TrimPrefix(v.Value, "?"))
		if err != nil {
			return nil, object.NewError(pos, "%s: %v", name, err)
		}
		return parsed, nil
	case *object.Hash:
		for _, pair := range v.Pairs {
			key, ok := pair.Key.(*object.String)
			if !ok {
				continue
			}
			values.Add(key.Value, pair.Value.Inspect())
		}
		return values, nil
	case *object.Array:
		for _, item := range v.Elements {
			entry, ok := item.(*object.Array)
			if !ok || len(entry.Elements) < 2 {
				return nil, object.NewError(pos, "%s: entries must be [name, value] arrays", name)
			}
			values.Add(entry.Elements[0].Inspect(), entry.Elements[1].Inspect())
		}
		return values, nil
	default:
		return nil, object.NewError(pos, "%s: init must be a string, object, array, null or undefined", name)
	}
}

func newURLSearchParamsObject(values url.Values, parent *object.Hash) *object.Hash {
	state := &urlSearchParamsState{values: cloneURLValues(values), parent: parent}
	obj := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(obj, "_state", &object.GoObject{Value: state})
	setHashMember(obj, "get", &object.Builtin{Name: "URLSearchParams.get", Fn: urlSearchParamsGet, Extra: &object.GoObject{Value: state}})
	setHashMember(obj, "has", &object.Builtin{Name: "URLSearchParams.has", Fn: urlSearchParamsHas, Extra: &object.GoObject{Value: state}})
	setHashMember(obj, "set", &object.Builtin{Name: "URLSearchParams.set", Fn: urlSearchParamsSet, Extra: &object.GoObject{Value: state}})
	setHashMember(obj, "append", &object.Builtin{Name: "URLSearchParams.append", Fn: urlSearchParamsAppend, Extra: &object.GoObject{Value: state}})
	setHashMember(obj, "delete", &object.Builtin{Name: "URLSearchParams.delete", Fn: urlSearchParamsDelete, Extra: &object.GoObject{Value: state}})
	setHashMember(obj, "toString", &object.Builtin{Name: "URLSearchParams.toString", Fn: urlSearchParamsToString, Extra: &object.GoObject{Value: state}})
	return obj
}

func cloneURLValues(values url.Values) url.Values {
	out := url.Values{}
	for key, vals := range values {
		copied := make([]string, len(vals))
		copy(copied, vals)
		out[key] = copied
	}
	return out
}

func urlSearchParamsStateFromExtra(pos ast.Position, name string, extra object.Object) (*urlSearchParamsState, *object.Error) {
	goObj, ok := extra.(*object.GoObject)
	if !ok {
		return nil, object.NewError(pos, "%s: missing URLSearchParams receiver", name)
	}
	state, ok := goObj.Value.(*urlSearchParamsState)
	if !ok {
		return nil, object.NewError(pos, "%s: invalid URLSearchParams receiver", name)
	}
	return state, nil
}

func urlSearchParamsGet(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	state, errObj := urlSearchParamsStateFromExtra(pos, "URLSearchParams.get", env.Extra)
	if errObj != nil {
		return errObj
	}
	name, errObj := requiredString(pos, "URLSearchParams.get", args, 0, "name")
	if errObj != nil {
		return errObj
	}
	if values, ok := state.values[name]; ok && len(values) > 0 {
		return &object.String{Value: values[0]}
	}
	return object.NULL
}

func urlSearchParamsHas(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	state, errObj := urlSearchParamsStateFromExtra(pos, "URLSearchParams.has", env.Extra)
	if errObj != nil {
		return errObj
	}
	name, errObj := requiredString(pos, "URLSearchParams.has", args, 0, "name")
	if errObj != nil {
		return errObj
	}
	_, ok := state.values[name]
	return object.NativeBool(ok)
}

func urlSearchParamsSet(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	state, errObj := urlSearchParamsStateFromExtra(pos, "URLSearchParams.set", env.Extra)
	if errObj != nil {
		return errObj
	}
	name, value, errObj := urlSearchParamsNameValue(pos, "URLSearchParams.set", args)
	if errObj != nil {
		return errObj
	}
	state.values.Set(name, value)
	state.sync()
	return object.UNDEFINED
}

func urlSearchParamsAppend(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	state, errObj := urlSearchParamsStateFromExtra(pos, "URLSearchParams.append", env.Extra)
	if errObj != nil {
		return errObj
	}
	name, value, errObj := urlSearchParamsNameValue(pos, "URLSearchParams.append", args)
	if errObj != nil {
		return errObj
	}
	state.values.Add(name, value)
	state.sync()
	return object.UNDEFINED
}

func urlSearchParamsDelete(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	state, errObj := urlSearchParamsStateFromExtra(pos, "URLSearchParams.delete", env.Extra)
	if errObj != nil {
		return errObj
	}
	name, errObj := requiredString(pos, "URLSearchParams.delete", args, 0, "name")
	if errObj != nil {
		return errObj
	}
	state.values.Del(name)
	state.sync()
	return object.UNDEFINED
}

func urlSearchParamsToString(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	state, errObj := urlSearchParamsStateFromExtra(pos, "URLSearchParams.toString", env.Extra)
	if errObj != nil {
		return errObj
	}
	return &object.String{Value: state.values.Encode()}
}

func urlSearchParamsNameValue(pos ast.Position, name string, args []object.Object) (string, string, *object.Error) {
	key, errObj := requiredString(pos, name, args, 0, "name")
	if errObj != nil {
		return "", "", errObj
	}
	if len(args) < 2 {
		return "", "", object.NewError(pos, "%s requires a value", name)
	}
	return key, args[1].Inspect(), nil
}

func (state *urlSearchParamsState) sync() {
	if state.parent == nil {
		return
	}
	query := state.values.Encode()
	search := ""
	if query != "" {
		search = "?" + query
	}
	setHashMember(state.parent, "search", &object.String{Value: search})
	u, err := urlFromObject(ast.Position{}, "URLSearchParams", state.parent)
	if err != nil {
		return
	}
	u.RawQuery = query
	populateURLObject(state.parent, u)
}
