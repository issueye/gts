package stdlib

import (
	"net"
	"net/http"
	"strings"
)

var hopByHopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

func copyHTTPHeader(dst, src http.Header) {
	for k, vals := range src {
		for _, v := range vals {
			dst.Add(k, v)
		}
	}
}

func removeHopByHopHeaders(header http.Header) {
	if connection := header.Get("Connection"); connection != "" {
		for _, name := range strings.Split(connection, ",") {
			if trimmed := strings.TrimSpace(name); trimmed != "" {
				header.Del(trimmed)
			}
		}
	}
	for _, name := range hopByHopHeaders {
		header.Del(name)
	}
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	default:
		return a + b
	}
}

func stripURLPathPrefix(path, prefix string) string {
	if prefix == "" || prefix == "/" {
		return path
	}
	prefix = "/" + strings.Trim(prefix, "/")
	if path == prefix {
		return "/"
	}
	if strings.HasPrefix(path, prefix+"/") {
		return strings.TrimPrefix(path, prefix)
	}
	return path
}

func appendForwardedFor(existing, remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil || host == "" {
		host = remoteAddr
	}
	if existing == "" {
		return host
	}
	return existing + ", " + host
}

func forwardedProto(r *http.Request) string {
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}
