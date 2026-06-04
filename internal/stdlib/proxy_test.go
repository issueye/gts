package stdlib

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/issueye/goscript/internal/ast"
	"github.com/issueye/goscript/internal/object"
)

func TestHTTPClientRequestUsesProxy(t *testing.T) {
	var proxiedURL string
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxiedURL = r.URL.String()
		w.Header().Set("X-Proxied", "yes")
		_, _ = w.Write([]byte("via proxy"))
	}))
	defer proxy.Close()

	opts := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(opts, "url", &object.String{Value: "http://upstream.test/data?x=1"})
	setHashMember(opts, "proxy", &object.String{Value: proxy.URL})

	result := httpClientRequest(object.NewEnvironment(), ast.Position{}, opts)
	resp, ok := result.(*object.Hash)
	if !ok {
		t.Fatalf("want response hash, got %T: %s", result, result.Inspect())
	}
	body, _ := hashValue(resp, "body")
	if body.Inspect() != "via proxy" {
		t.Fatalf("want proxy response body, got %q", body.Inspect())
	}
	if proxiedURL != "http://upstream.test/data?x=1" {
		t.Fatalf("request did not use proxy, saw URL %q", proxiedURL)
	}
}

func TestWebProxyForwardsRequest(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPost {
			t.Fatalf("want POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/users" || r.URL.RawQuery != "q=1" {
			t.Fatalf("unexpected upstream URL: %s", r.URL.String())
		}
		if string(data) != "payload" {
			t.Fatalf("want forwarded body, got %q", string(data))
		}
		if r.Header.Get("X-Forwarded-Host") == "" || r.Header.Get("X-Forwarded-For") == "" {
			t.Fatalf("missing forwarded headers: %#v", r.Header)
		}
		w.Header().Set("X-Upstream", "ok")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("forwarded"))
	}))
	defer upstream.Close()

	app := &webApp{}
	proxy := webProxy(object.NewEnvironment(), ast.Position{}, proxyOptions(upstream.URL, "/api"))
	app.addRoute("USE", "/api", []object.Object{proxy})
	server := httptest.NewServer(app)
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/users?q=1", "text/plain", strings.NewReader("payload"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("want 201, got %d: %s", resp.StatusCode, string(data))
	}
	if string(data) != "forwarded" {
		t.Fatalf("want forwarded response body, got %q", string(data))
	}
	if resp.Header.Get("X-Upstream") != "ok" {
		t.Fatalf("missing upstream response header")
	}
}

func proxyOptions(target, stripPrefix string) *object.Hash {
	opts := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
	setHashMember(opts, "target", &object.String{Value: target + "/v1"})
	setHashMember(opts, "stripPrefix", &object.String{Value: stripPrefix})
	return opts
}
