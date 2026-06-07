package imbot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/issueye/goscript/sdk/gtp"
)

const ModuleName = "@plugin/im-bot"

type Service struct {
	mu       sync.Mutex
	adapters map[string]Adapter
	client   *http.Client
}

type Adapter interface {
	Platform() string
	Send(req SendRequest) (gtp.Value, error)
	Info() gtp.Value
}

type SendRequest struct {
	To      string
	ToType  string
	Text    string
	MsgType string
	Extra   gtp.Value
}

func NewService() *Service {
	return &Service{
		adapters: make(map[string]Adapter),
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *Service) Run(in io.Reader, out io.Writer) error {
	decoder := gtp.NewDecoder(in)
	encoder := gtp.NewEncoder(out)
	for {
		frame, err := decoder.Decode()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if frame.Type == "hello" {
			if err := encoder.Encode(ReadyFrame(frame.ID)); err != nil {
				return err
			}
			continue
		}
		if err := encoder.Encode(s.Handle(frame)); err != nil {
			return err
		}
	}
}

func ReadyFrame(id string) gtp.Frame {
	return gtp.Frame{
		Version:      gtp.Version,
		ID:           id,
		Type:         "ready",
		Service:      "im-bot",
		Capabilities: []string{"call"},
		Modules: map[string][]string{
			ModuleName: {"configure", "list", "send"},
		},
	}
}

func (s *Service) Handle(frame gtp.Frame) gtp.Frame {
	if frame.Type != "call" {
		return gtp.ErrorResult(frame.ID, gtp.TypeError("im-bot expects call frames"))
	}
	if frame.Module != "" && frame.Module != ModuleName {
		return gtp.ErrorResult(frame.ID, gtp.NotFoundError("unknown module %s", frame.Module))
	}
	switch frame.Method {
	case "configure":
		return s.handleConfigure(frame)
	case "list":
		return s.handleList(frame)
	case "send":
		return s.handleSend(frame)
	default:
		return gtp.ErrorResult(frame.ID, gtp.NotFoundError("unknown im-bot method %s", frame.Method))
	}
}

func (s *Service) handleConfigure(frame gtp.Frame) gtp.Frame {
	opts, errObj := gtp.RequiredObjectArg(frame.Args, 0, "options")
	if errObj != nil {
		return gtp.ErrorResult(frame.ID, errObj)
	}
	name, ok := gtp.StringField(opts, "name")
	if !ok || strings.TrimSpace(name) == "" {
		return gtp.ErrorResult(frame.ID, gtp.TypeError("options.name is required"))
	}
	platform, ok := gtp.StringField(opts, "platform")
	if !ok || strings.TrimSpace(platform) == "" {
		return gtp.ErrorResult(frame.ID, gtp.TypeError("options.platform is required"))
	}
	adapter, err := s.newAdapter(strings.ToLower(strings.TrimSpace(platform)), opts)
	if err != nil {
		return gtp.ErrorResult(frame.ID, gtp.TypeError("%v", err))
	}
	s.mu.Lock()
	s.adapters[name] = adapter
	s.mu.Unlock()
	return gtp.OKResult(frame.ID, adapter.Info())
}

func (s *Service) handleList(frame gtp.Frame) gtp.Frame {
	s.mu.Lock()
	items := make([]gtp.Value, 0, len(s.adapters))
	for name, adapter := range s.adapters {
		info := adapter.Info()
		if info.Type == "object" {
			info.Fields["name"] = gtp.String(name)
		}
		items = append(items, info)
	}
	s.mu.Unlock()
	return gtp.OKResult(frame.ID, gtp.Array(items))
}

func (s *Service) handleSend(frame gtp.Frame) gtp.Frame {
	opts, errObj := gtp.RequiredObjectArg(frame.Args, 0, "options")
	if errObj != nil {
		return gtp.ErrorResult(frame.ID, errObj)
	}
	name, ok := gtp.StringField(opts, "adapter")
	if !ok || strings.TrimSpace(name) == "" {
		return gtp.ErrorResult(frame.ID, gtp.TypeError("options.adapter is required"))
	}
	req := SendRequest{}
	req.To, _ = gtp.StringField(opts, "to")
	req.ToType, _ = gtp.StringField(opts, "toType")
	req.Text, _ = gtp.StringField(opts, "text")
	req.MsgType, _ = gtp.StringField(opts, "msgType")
	req.Extra, _ = gtp.Field(opts, "extra")
	if req.Text == "" {
		req.Text, _ = gtp.StringField(opts, "message")
	}
	if strings.TrimSpace(req.To) == "" {
		return gtp.ErrorResult(frame.ID, gtp.TypeError("options.to is required"))
	}
	if strings.TrimSpace(req.Text) == "" {
		return gtp.ErrorResult(frame.ID, gtp.TypeError("options.text is required"))
	}

	s.mu.Lock()
	adapter := s.adapters[name]
	s.mu.Unlock()
	if adapter == nil {
		return gtp.ErrorResult(frame.ID, gtp.NotFoundError("adapter %s is not configured", name))
	}
	result, err := adapter.Send(req)
	if err != nil {
		return gtp.ErrorResult(frame.ID, gtp.HostError("%v", err))
	}
	return gtp.OKResult(frame.ID, result)
}

func (s *Service) newAdapter(platform string, opts gtp.Value) (Adapter, error) {
	switch platform {
	case "feishu", "lark":
		return newFeishuAdapter(platform, opts, s.client)
	case "onebot", "qq":
		return newOneBotAdapter(opts, s.client)
	case "qqbot":
		return newQQBotAdapter(opts, s.client)
	case "weixin", "wechat-personal":
		return newWeixinAdapter(opts, s.client)
	default:
		return nil, fmt.Errorf("unsupported platform %q", platform)
	}
}

func httpJSON(client *http.Client, method, rawURL string, headers map[string]string, body any) (gtp.Value, int, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return gtp.Null(), 0, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, rawURL, reader)
	if err != nil {
		return gtp.Null(), 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return gtp.Null(), 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return gtp.Null(), resp.StatusCode, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return gtp.Null(), resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return gtp.Object(map[string]gtp.Value{"status": gtp.Number(float64(resp.StatusCode))}), resp.StatusCode, nil
	}
	value, err := decodeJSONValue(data)
	return value, resp.StatusCode, err
}

func decodeJSONValue(data []byte) (gtp.Value, error) {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return gtp.Null(), err
	}
	return jsonToGTP(raw), nil
}

func jsonToGTP(raw any) gtp.Value {
	switch v := raw.(type) {
	case nil:
		return gtp.Null()
	case bool:
		return gtp.Bool(v)
	case float64:
		return gtp.Number(v)
	case string:
		return gtp.String(v)
	case []any:
		items := make([]gtp.Value, len(v))
		for i, item := range v {
			items[i] = jsonToGTP(item)
		}
		return gtp.Array(items)
	case map[string]any:
		fields := make(map[string]gtp.Value, len(v))
		for key, item := range v {
			fields[key] = jsonToGTP(item)
		}
		return gtp.Object(fields)
	default:
		return gtp.String(fmt.Sprint(v))
	}
}

func boolField(obj gtp.Value, key string) (bool, bool) {
	value, ok := gtp.Field(obj, key)
	if !ok {
		return false, false
	}
	return value.BoolValue()
}

func stringDefault(obj gtp.Value, key, fallback string) string {
	if value, ok := gtp.StringField(obj, key); ok && value != "" {
		return value
	}
	return fallback
}

func numberString(raw string) any {
	if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return n
	}
	return raw
}

func joinURL(base, path string) string {
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(path, "/")
}

func queryURL(base string, query map[string]string) string {
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	q := u.Query()
	for key, value := range query {
		if value != "" {
			q.Set(key, value)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}
