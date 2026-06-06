package imbot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/issueye/goscript/internal/gtp"
)

type feishuAdapter struct {
	platform  string
	appID     string
	appSecret string
	baseURL   string
	client    *http.Client

	mu          sync.Mutex
	token       string
	tokenExpiry time.Time
}

func newFeishuAdapter(platform string, opts gtp.Value, client *http.Client) (Adapter, error) {
	appID, _ := gtp.StringField(opts, "appId")
	if appID == "" {
		appID, _ = gtp.StringField(opts, "app_id")
	}
	appSecret, _ := gtp.StringField(opts, "appSecret")
	if appSecret == "" {
		appSecret, _ = gtp.StringField(opts, "app_secret")
	}
	if appID == "" || appSecret == "" {
		return nil, fmt.Errorf("feishu requires appId/appSecret")
	}
	return &feishuAdapter{
		platform:  platform,
		appID:     appID,
		appSecret: appSecret,
		baseURL:   stringDefault(opts, "baseUrl", "https://open.feishu.cn"),
		client:    client,
	}, nil
}

func (a *feishuAdapter) Platform() string { return a.platform }

func (a *feishuAdapter) Info() gtp.Value {
	return gtp.Object(map[string]gtp.Value{
		"platform": gtp.String(a.platform),
		"baseUrl":  gtp.String(a.baseURL),
	})
}

func (a *feishuAdapter) Send(req SendRequest) (gtp.Value, error) {
	token, err := a.accessToken()
	if err != nil {
		return gtp.Null(), err
	}
	toType := req.ToType
	if toType == "" {
		toType = "chat_id"
	}
	msgType := req.MsgType
	if msgType == "" {
		msgType = "text"
	}
	content := map[string]string{"text": req.Text}
	contentJSON, _ := jsonString(content)
	body := map[string]any{
		"receive_id": req.To,
		"msg_type":   msgType,
		"content":    contentJSON,
	}
	u := queryURL(joinURL(a.baseURL, "/open-apis/im/v1/messages"), map[string]string{"receive_id_type": toType})
	result, _, err := httpJSON(a.client, http.MethodPost, u, map[string]string{"Authorization": "Bearer " + token}, body)
	return result, err
}

func (a *feishuAdapter) accessToken() (string, error) {
	a.mu.Lock()
	if a.token != "" && time.Now().Before(a.tokenExpiry.Add(-5*time.Minute)) {
		token := a.token
		a.mu.Unlock()
		return token, nil
	}
	a.mu.Unlock()

	body := map[string]string{"app_id": a.appID, "app_secret": a.appSecret}
	result, _, err := httpJSON(a.client, http.MethodPost, joinURL(a.baseURL, "/open-apis/auth/v3/tenant_access_token/internal"), nil, body)
	if err != nil {
		return "", err
	}
	token, _ := gtp.StringField(result, "tenant_access_token")
	if token == "" {
		token, _ = gtp.StringField(result, "app_access_token")
	}
	if token == "" {
		return "", fmt.Errorf("feishu token response did not include access token")
	}
	expire := 7200.0
	if n, ok := gtp.NumberField(result, "expire"); ok && n > 0 {
		expire = n
	}
	a.mu.Lock()
	a.token = token
	a.tokenExpiry = time.Now().Add(time.Duration(expire) * time.Second)
	a.mu.Unlock()
	return token, nil
}

type oneBotAdapter struct {
	baseURL string
	token   string
	client  *http.Client
}

func newOneBotAdapter(opts gtp.Value, client *http.Client) (Adapter, error) {
	baseURL, _ := gtp.StringField(opts, "baseUrl")
	if baseURL == "" {
		baseURL, _ = gtp.StringField(opts, "httpUrl")
	}
	if baseURL == "" {
		return nil, fmt.Errorf("onebot requires baseUrl")
	}
	token, _ := gtp.StringField(opts, "token")
	return &oneBotAdapter{baseURL: baseURL, token: token, client: client}, nil
}

func (a *oneBotAdapter) Platform() string { return "onebot" }
func (a *oneBotAdapter) Info() gtp.Value {
	return gtp.Object(map[string]gtp.Value{"platform": gtp.String("onebot"), "baseUrl": gtp.String(a.baseURL), "tokenSet": gtp.Bool(a.token != "")})
}

func (a *oneBotAdapter) Send(req SendRequest) (gtp.Value, error) {
	action := "send_private_msg"
	body := map[string]any{"message": req.Text, "auto_escape": true}
	if req.ToType == "group" || strings.HasPrefix(req.To, "group:") {
		action = "send_group_msg"
		body["group_id"] = numberString(strings.TrimPrefix(req.To, "group:"))
	} else {
		body["user_id"] = numberString(strings.TrimPrefix(req.To, "private:"))
	}
	headers := map[string]string{}
	if a.token != "" {
		headers["Authorization"] = "Bearer " + a.token
	}
	result, _, err := httpJSON(a.client, http.MethodPost, joinURL(a.baseURL, action), headers, body)
	return result, err
}

type qqBotAdapter struct {
	appID     string
	appSecret string
	apiBase   string
	tokenURL  string
	client    *http.Client

	mu          sync.Mutex
	token       string
	tokenExpiry time.Time
}

func newQQBotAdapter(opts gtp.Value, client *http.Client) (Adapter, error) {
	appID, _ := gtp.StringField(opts, "appId")
	if appID == "" {
		appID, _ = gtp.StringField(opts, "app_id")
	}
	appSecret, _ := gtp.StringField(opts, "appSecret")
	if appSecret == "" {
		appSecret, _ = gtp.StringField(opts, "app_secret")
	}
	if appID == "" || appSecret == "" {
		return nil, fmt.Errorf("qqbot requires appId/appSecret")
	}
	apiBase := stringDefault(opts, "apiBase", "https://api.sgroup.qq.com")
	if sandbox, ok := boolField(opts, "sandbox"); ok && sandbox {
		apiBase = "https://sandbox.api.sgroup.qq.com"
	}
	return &qqBotAdapter{
		appID: appID, appSecret: appSecret, apiBase: apiBase,
		tokenURL: stringDefault(opts, "tokenUrl", "https://bots.qq.com/app/getAppAccessToken"),
		client:   client,
	}, nil
}

func (a *qqBotAdapter) Platform() string { return "qqbot" }
func (a *qqBotAdapter) Info() gtp.Value {
	return gtp.Object(map[string]gtp.Value{"platform": gtp.String("qqbot"), "apiBase": gtp.String(a.apiBase)})
}

func (a *qqBotAdapter) Send(req SendRequest) (gtp.Value, error) {
	token, err := a.accessToken()
	if err != nil {
		return gtp.Null(), err
	}
	msgType := 0
	body := map[string]any{"msg_type": msgType, "content": req.Text}
	var path string
	if req.ToType == "group" || strings.HasPrefix(req.To, "group:") {
		path = "/v2/groups/" + strings.TrimPrefix(req.To, "group:") + "/messages"
	} else {
		path = "/v2/users/" + strings.TrimPrefix(req.To, "c2c:") + "/messages"
	}
	result, _, err := httpJSON(a.client, http.MethodPost, joinURL(a.apiBase, path), map[string]string{"Authorization": "QQBot " + token}, body)
	return result, err
}

func (a *qqBotAdapter) accessToken() (string, error) {
	a.mu.Lock()
	if a.token != "" && time.Now().Before(a.tokenExpiry.Add(-5*time.Minute)) {
		token := a.token
		a.mu.Unlock()
		return token, nil
	}
	a.mu.Unlock()
	result, _, err := httpJSON(a.client, http.MethodPost, a.tokenURL, nil, map[string]string{"appId": a.appID, "clientSecret": a.appSecret})
	if err != nil {
		return "", err
	}
	token, _ := gtp.StringField(result, "access_token")
	if token == "" {
		return "", fmt.Errorf("qqbot token response did not include access_token")
	}
	expires := 7200.0
	if s, ok := gtp.StringField(result, "expires_in"); ok && s != "" {
		fmt.Sscanf(s, "%f", &expires)
	} else if n, ok := gtp.NumberField(result, "expires_in"); ok {
		expires = n
	}
	a.mu.Lock()
	a.token = token
	a.tokenExpiry = time.Now().Add(time.Duration(expires) * time.Second)
	a.mu.Unlock()
	return token, nil
}

type weixinAdapter struct {
	baseURL string
	token   string
	client  *http.Client
}

func newWeixinAdapter(opts gtp.Value, client *http.Client) (Adapter, error) {
	token, _ := gtp.StringField(opts, "token")
	if token == "" {
		return nil, fmt.Errorf("weixin requires token")
	}
	return &weixinAdapter{
		baseURL: stringDefault(opts, "baseUrl", "https://ilinkai.weixin.qq.com"),
		token:   token,
		client:  client,
	}, nil
}

func (a *weixinAdapter) Platform() string { return "weixin" }
func (a *weixinAdapter) Info() gtp.Value {
	return gtp.Object(map[string]gtp.Value{"platform": gtp.String("weixin"), "baseUrl": gtp.String(a.baseURL), "tokenSet": gtp.Bool(a.token != "")})
}

func (a *weixinAdapter) Send(req SendRequest) (gtp.Value, error) {
	contextToken := ""
	if req.Extra.Type == "object" {
		contextToken, _ = gtp.StringField(req.Extra, "contextToken")
		if contextToken == "" {
			contextToken, _ = gtp.StringField(req.Extra, "context_token")
		}
	}
	body := map[string]any{
		"to_user_id":     req.To,
		"content":        req.Text,
		"context_token":  contextToken,
		"client_msg_id":  fmt.Sprintf("gts-%d", time.Now().UnixNano()),
		"message_format": "text",
	}
	result, _, err := httpJSON(a.client, http.MethodPost, joinURL(a.baseURL, "/ilink/bot/sendmessage"), map[string]string{"Authorization": "Bearer " + a.token}, body)
	return result, err
}

func jsonString(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
