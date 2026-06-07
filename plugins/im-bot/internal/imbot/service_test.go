package imbot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/issueye/goscript/sdk/gtp"
)

func TestConfigureAndList(t *testing.T) {
	svc := NewService()
	result := svc.Handle(gtp.Frame{ID: "1", Type: "call", Module: ModuleName, Method: "configure", Args: []gtp.Value{
		gtp.Object(map[string]gtp.Value{
			"name":     gtp.String("qq"),
			"platform": gtp.String("onebot"),
			"baseUrl":  gtp.String("http://127.0.0.1:5700"),
			"token":    gtp.String("secret"),
		}),
	}})
	if result.OK == nil || !*result.OK {
		t.Fatalf("configure = %#v", result)
	}
	list := svc.Handle(gtp.Frame{ID: "2", Type: "call", Module: ModuleName, Method: "list"})
	if list.Result == nil || len(list.Result.Items) != 1 {
		t.Fatalf("list = %#v", list)
	}
}

func TestOneBotSend(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "retcode": 0, "data": map[string]any{"message_id": 7}})
	}))
	defer server.Close()

	svc := NewService()
	configure(t, svc, map[string]gtp.Value{
		"name":     gtp.String("qq"),
		"platform": gtp.String("onebot"),
		"baseUrl":  gtp.String(server.URL),
		"token":    gtp.String("secret"),
	})
	send := svc.Handle(gtp.Frame{ID: "2", Type: "call", Module: ModuleName, Method: "send", Args: []gtp.Value{gtp.Object(map[string]gtp.Value{
		"adapter": gtp.String("qq"),
		"to":      gtp.String("group:10001"),
		"toType":  gtp.String("group"),
		"text":    gtp.String("hello"),
	})}})
	if send.OK == nil || !*send.OK {
		t.Fatalf("send = %#v", send)
	}
	if gotPath != "/send_group_msg" || gotAuth != "Bearer secret" || gotBody["message"] != "hello" {
		t.Fatalf("path=%s auth=%s body=%#v", gotPath, gotAuth, gotBody)
	}
}

func TestFeishuSend(t *testing.T) {
	var sawMessage bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/auth/v3/tenant_access_token/internal":
			_ = json.NewEncoder(w).Encode(map[string]any{"tenant_access_token": "tenant-token", "expire": 7200})
		case "/open-apis/im/v1/messages":
			sawMessage = true
			if r.Header.Get("Authorization") != "Bearer tenant-token" {
				t.Fatalf("auth = %s", r.Header.Get("Authorization"))
			}
			if r.URL.Query().Get("receive_id_type") != "chat_id" {
				t.Fatalf("query = %s", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"message_id": "om_x"}})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	svc := NewService()
	configure(t, svc, map[string]gtp.Value{
		"name":      gtp.String("fs"),
		"platform":  gtp.String("feishu"),
		"appId":     gtp.String("app"),
		"appSecret": gtp.String("secret"),
		"baseUrl":   gtp.String(server.URL),
	})
	send := svc.Handle(gtp.Frame{ID: "2", Type: "call", Module: ModuleName, Method: "send", Args: []gtp.Value{gtp.Object(map[string]gtp.Value{
		"adapter": gtp.String("fs"),
		"to":      gtp.String("oc_x"),
		"text":    gtp.String("hello"),
	})}})
	if send.OK == nil || !*send.OK || !sawMessage {
		t.Fatalf("send = %#v saw=%v", send, sawMessage)
	}
}

func TestQQBotSend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "qq-token", "expires_in": "7200"})
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/v2/groups/") {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "QQBot qq-token" {
			t.Fatalf("auth = %s", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "msg"})
	}))
	defer server.Close()

	svc := NewService()
	configure(t, svc, map[string]gtp.Value{
		"name":      gtp.String("qqbot"),
		"platform":  gtp.String("qqbot"),
		"appId":     gtp.String("app"),
		"appSecret": gtp.String("secret"),
		"apiBase":   gtp.String(server.URL),
		"tokenUrl":  gtp.String(server.URL + "/token"),
	})
	send := svc.Handle(gtp.Frame{ID: "2", Type: "call", Module: ModuleName, Method: "send", Args: []gtp.Value{gtp.Object(map[string]gtp.Value{
		"adapter": gtp.String("qqbot"),
		"to":      gtp.String("group:gid"),
		"toType":  gtp.String("group"),
		"text":    gtp.String("hello"),
	})}})
	if send.OK == nil || !*send.OK {
		t.Fatalf("send = %#v", send)
	}
}

func TestWeixinSend(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ilink/bot/sendmessage" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer wx-token" {
			t.Fatalf("auth = %s", r.Header.Get("Authorization"))
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(map[string]any{"ret": 0})
	}))
	defer server.Close()

	svc := NewService()
	configure(t, svc, map[string]gtp.Value{
		"name":     gtp.String("wx"),
		"platform": gtp.String("weixin"),
		"token":    gtp.String("wx-token"),
		"baseUrl":  gtp.String(server.URL),
	})
	send := svc.Handle(gtp.Frame{ID: "2", Type: "call", Module: ModuleName, Method: "send", Args: []gtp.Value{gtp.Object(map[string]gtp.Value{
		"adapter": gtp.String("wx"),
		"to":      gtp.String("u@im.wechat"),
		"text":    gtp.String("hello"),
		"extra":   gtp.Object(map[string]gtp.Value{"contextToken": gtp.String("ctx")}),
	})}})
	if send.OK == nil || !*send.OK || got["context_token"] != "ctx" {
		t.Fatalf("send = %#v body=%#v", send, got)
	}
}

func configure(t *testing.T, svc *Service, fields map[string]gtp.Value) {
	t.Helper()
	result := svc.Handle(gtp.Frame{ID: "1", Type: "call", Module: ModuleName, Method: "configure", Args: []gtp.Value{gtp.Object(fields)}})
	if result.OK == nil || !*result.OK {
		t.Fatalf("configure = %#v", result)
	}
}
