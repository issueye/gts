package stdlib

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func httpClientGet(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	opts, errObj := parseHTTPGetOptions(pos, args)
	if errObj != nil {
		return errObj
	}
	client := newHTTPClient(opts)
	req, err := http.NewRequest("GET", opts.url, nil)
	if err != nil {
		return object.NewError(pos, "http.get: %v", err)
	}
	for k, v := range opts.headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return object.NewError(pos, "http.get: %v", err)
	}
	defer resp.Body.Close()
	return buildResponseObject(resp)
}

func httpClientPost(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	opts, contentType, errObj := parseHTTPPostOptions(pos, args)
	if errObj != nil {
		return errObj
	}
	client := newHTTPClient(opts)
	req, err := http.NewRequest("POST", opts.url, opts.body)
	if err != nil {
		return object.NewError(pos, "http.post: %v", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range opts.headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return object.NewError(pos, "http.post: %v", err)
	}
	defer resp.Body.Close()
	return buildResponseObject(resp)
}

func httpClientRequest(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	opts, errObj := parseHTTPRequestOptions(pos, "http.request", args)
	if errObj != nil {
		return errObj
	}
	client := newHTTPClient(opts)
	req, err := http.NewRequest(opts.method, opts.url, opts.body)
	if err != nil {
		return object.NewError(pos, "http.request: %v", err)
	}
	for k, v := range opts.headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return object.NewError(pos, "http.request: %v", err)
	}
	if opts.stream {
		return buildStreamingResponseObject(resp)
	}
	defer resp.Body.Close()
	return buildResponseObject(resp)
}

func httpClientStream(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
	opts, errObj := parseHTTPRequestOptions(pos, "http.stream", args)
	if errObj != nil {
		return errObj
	}
	client := newHTTPClient(opts)
	req, err := http.NewRequest(opts.method, opts.url, opts.body)
	if err != nil {
		return object.NewError(pos, "http.stream: %v", err)
	}
	for k, v := range opts.headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return object.NewError(pos, "http.stream: %v", err)
	}
	return buildStreamingResponseObject(resp)
}

func buildResponseObject(resp *http.Response) object.Object {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		bodyBytes = []byte{}
	}
	result := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(result, "status", &object.Number{Value: float64(resp.StatusCode)})
	setHashMember(result, "statusText", &object.String{Value: resp.Status})
	setHashMember(result, "body", &object.String{Value: string(bodyBytes)})
	setHashMember(result, "ok", object.NativeBool(resp.StatusCode >= 200 && resp.StatusCode < 300))
	headers := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	for k, vals := range resp.Header {
		if len(vals) > 0 {
			setHashMember(headers, k, &object.String{Value: vals[0]})
		}
	}
	setHashMember(result, "headers", headers)
	setHashMember(result, "contentLength", &object.Number{Value: float64(resp.ContentLength)})
	return result
}

func buildStreamingResponseObject(resp *http.Response) object.Object {
	result := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(result, "status", &object.Number{Value: float64(resp.StatusCode)})
	setHashMember(result, "statusText", &object.String{Value: resp.Status})
	setHashMember(result, "ok", object.NativeBool(resp.StatusCode >= 200 && resp.StatusCode < 300))
	headers := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	for k, vals := range resp.Header {
		if len(vals) > 0 {
			setHashMember(headers, k, &object.String{Value: vals[0]})
		}
	}
	setHashMember(result, "headers", headers)
	setHashMember(result, "contentLength", &object.Number{Value: float64(resp.ContentLength)})
	setHashMember(result, "body", newReadableStream(resp.Body, resp.Body))
	setHashMember(result, "close", &object.Builtin{Name: "http.response.close", Fn: func(env *object.Environment, pos ast.Position, args ...object.Object) object.Object {
		if err := resp.Body.Close(); err != nil {
			return object.NewError(pos, "http.response.close: %v", err)
		}
		return object.UNDEFINED
	}})
	return result
}

type httpRequestOptions struct {
	method    string
	url       string
	headers   map[string]string
	body      io.Reader
	timeoutMs int
	stream    bool
	proxyURL  *url.URL
}

func newHTTPClient(opts *httpRequestOptions) *http.Client {
	client := &http.Client{}
	if opts.proxyURL != nil {
		client.Transport = &http.Transport{Proxy: http.ProxyURL(opts.proxyURL)}
	}
	if opts.timeoutMs > 0 {
		client.Timeout = time.Duration(opts.timeoutMs) * time.Millisecond
	}
	return client
}

func parseHTTPGetOptions(pos ast.Position, args []object.Object) (*httpRequestOptions, *object.Error) {
	if len(args) < 1 {
		return nil, object.NewError(pos, "http.get requires a URL or options object")
	}
	if _, ok := args[0].(*object.Hash); ok {
		opts, errObj := parseHTTPRequestOptions(pos, "http.get", args[:1])
		if errObj != nil {
			return nil, errObj
		}
		opts.method = "GET"
		return opts, nil
	}
	opts, errObj := parseHTTPRequestOptions(pos, "http.get", args[:1])
	if errObj != nil {
		return nil, errObj
	}
	if len(args) >= 2 {
		if errObj := mergeHTTPClientOptions(pos, "http.get", opts, args[1], true); errObj != nil {
			return nil, errObj
		}
	}
	return opts, nil
}

func parseHTTPPostOptions(pos ast.Position, args []object.Object) (*httpRequestOptions, string, *object.Error) {
	if len(args) < 1 {
		return nil, "", object.NewError(pos, "http.post requires a URL or options object")
	}
	if _, ok := args[0].(*object.Hash); ok {
		opts, errObj := parseHTTPRequestOptions(pos, "http.post", args[:1])
		if errObj != nil {
			return nil, "", errObj
		}
		opts.method = "POST"
		return opts, "", nil
	}
	opts, errObj := parseHTTPRequestOptions(pos, "http.post", args[:1])
	if errObj != nil {
		return nil, "", errObj
	}
	opts.method = "POST"
	contentType := ""
	if len(args) >= 2 {
		opts.body, contentType = httpBodyReader(args[1])
	}
	if len(args) >= 3 {
		if errObj := mergeHTTPClientOptions(pos, "http.post", opts, args[2], true); errObj != nil {
			return nil, "", errObj
		}
	}
	return opts, contentType, nil
}

func parseHTTPRequestOptions(pos ast.Position, name string, args []object.Object) (*httpRequestOptions, *object.Error) {
	if len(args) < 1 {
		return nil, object.NewError(pos, "%s requires an options object or URL string", name)
	}
	opts := &httpRequestOptions{method: "GET", headers: make(map[string]string)}
	if urlArg, ok := args[0].(*object.String); ok {
		opts.url = urlArg.Value
	} else if h, ok := args[0].(*object.Hash); ok {
		if v, ok := hashValue(h, "url"); ok {
			opts.url = v.Inspect()
		}
		if v, ok := hashValue(h, "method"); ok {
			opts.method = strings.ToUpper(v.Inspect())
		}
		if v, ok := hashValue(h, "headers"); ok {
			if headers, ok := v.(*object.Hash); ok {
				for _, pair := range headers.OrderedPairs() {
					opts.headers[pair.Key.Inspect()] = pair.Value.Inspect()
				}
			}
		}
		if v, ok := hashValue(h, "body"); ok {
			opts.body, _ = httpBodyReader(v)
		}
		if v, ok := hashValue(h, "timeoutMs"); ok {
			if n, ok := v.(*object.Number); ok {
				opts.timeoutMs = int(n.Value)
			}
		}
		if v, ok := hashValue(h, "stream"); ok {
			if b, ok := v.(*object.Boolean); ok {
				opts.stream = b.Value
			}
		}
		if v, ok := hashValue(h, "responseType"); ok && v.Inspect() == "stream" {
			opts.stream = true
		}
		if errObj := mergeHTTPClientOptions(pos, name, opts, h, false); errObj != nil {
			return nil, errObj
		}
	} else {
		return nil, object.NewError(pos, "%s: first argument must be a string URL or options object", name)
	}
	if opts.url == "" {
		return nil, object.NewError(pos, "%s: URL is required", name)
	}
	return opts, nil
}

func mergeHTTPClientOptions(pos ast.Position, name string, opts *httpRequestOptions, opt object.Object, fallbackToHeaders bool) *object.Error {
	h, ok := opt.(*object.Hash)
	if !ok {
		return nil
	}
	if headersObj, ok := hashValue(h, "headers"); ok {
		headers, ok := headersObj.(*object.Hash)
		if !ok {
			return object.NewError(pos, "%s: headers must be an object", name)
		}
		mergeHTTPHeaders(opts, headers)
	} else if fallbackToHeaders {
		mergeHTTPHeaders(opts, h)
	}
	if v, ok := hashValue(h, "timeoutMs"); ok {
		if n, ok := v.(*object.Number); ok {
			opts.timeoutMs = int(n.Value)
		}
	}
	if v, ok := hashValue(h, "proxy"); ok {
		proxyURL, err := url.Parse(v.Inspect())
		if err != nil || proxyURL.Scheme == "" || proxyURL.Host == "" {
			return object.NewError(pos, "%s: proxy must be an absolute URL", name)
		}
		opts.proxyURL = proxyURL
	}
	return nil
}

func mergeHTTPHeaders(opts *httpRequestOptions, headers *object.Hash) {
	for _, pair := range headers.OrderedPairs() {
		key := pair.Key.Inspect()
		if isHTTPClientOptionKey(key) {
			continue
		}
		opts.headers[key] = pair.Value.Inspect()
	}
}

func isHTTPClientOptionKey(key string) bool {
	switch key {
	case "url", "method", "headers", "body", "timeoutMs", "stream", "responseType", "proxy":
		return true
	default:
		return false
	}
}

func httpBodyReader(val object.Object) (io.Reader, string) {
	switch b := val.(type) {
	case *object.String:
		return strings.NewReader(b.Value), "text/plain"
	case *object.Hash:
		return strings.NewReader(toJSONString(b)), "application/json"
	default:
		return strings.NewReader(val.Inspect()), "text/plain"
	}
}

func toJSONString(hash *object.Hash) string {
	var buf bytes.Buffer
	buf.WriteByte('{')
	i := 0
	for _, pair := range hash.OrderedPairs() {
		if i > 0 {
			buf.WriteByte(',')
		}
		keyStr := pair.Key.Inspect()
		buf.WriteByte('"')
		buf.WriteString(keyStr)
		buf.WriteString(`":`)
		writeJSONValue(&buf, pair.Value)
		i++
	}
	buf.WriteByte('}')
	return buf.String()
}

func writeJSONValue(buf *bytes.Buffer, val object.Object) {
	switch v := val.(type) {
	case *object.String:
		buf.WriteByte('"')
		buf.WriteString(v.Value)
		buf.WriteByte('"')
	case *object.Number:
		buf.WriteString(v.Inspect())
	case *object.Boolean:
		buf.WriteString(v.Inspect())
	case *object.Null:
		buf.WriteString("null")
	case *object.Hash:
		buf.WriteString(toJSONString(v))
	case *object.Array:
		buf.WriteByte('[')
		for j, e := range v.Elements {
			if j > 0 {
				buf.WriteByte(',')
			}
			writeJSONValue(buf, e)
		}
		buf.WriteByte(']')
	default:
		buf.WriteByte('"')
		buf.WriteString(v.Inspect())
		buf.WriteByte('"')
	}
}
