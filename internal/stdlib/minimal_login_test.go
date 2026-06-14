package stdlib

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestMinimalLoginFlow tests a minimal login flow to isolate the issue.
func TestMinimalLoginFlow(t *testing.T) {
	src := `
let web = require("@std/web");
let shared = require("@std/shared");
let app = web.createApp({ concurrency: "isolated", poolSize: 2 });
let users = shared.map("minimal-users");

app.post("/register", function(req, res) {
  let data = JSON.parse(req.body);
  users.set(data.username, JSON.stringify({ password: data.password }));
  res.json({ ok: true });
});

app.post("/login", function(req, res) {
  let data = JSON.parse(req.body);
  let userJson = users.get(data.username);
  
  if (userJson === undefined) {
    res.status(401).json({ error: "not found" });
    return;
  }
  
  let user = JSON.parse(userJson);
  if (user.password !== data.password) {
    res.status(401).json({ error: "bad password" });
    return;
  }
  
  let token = data.username + "-token";
  res.json({ token: token });
});

let server = app.listen(0);
app;
`
	app := evalWebIsolatedApp(t, src)
	server := httptest.NewServer(app)
	defer server.Close()

	// Register
	regBody, _ := json.Marshal(map[string]string{"username": "test", "password": "pass"})
	resp, err := http.Post(server.URL+"/register", "application/json", bytes.NewReader(regBody))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	t.Logf("POST /register: %d %s", resp.StatusCode, body)
	if resp.StatusCode != 200 {
		t.Fatalf("register failed")
	}

	// Login
	loginBody, _ := json.Marshal(map[string]string{"username": "test", "password": "pass"})
	resp, err = http.Post(server.URL+"/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	t.Logf("POST /login: %d %s", resp.StatusCode, body)
	
	if resp.StatusCode != 200 {
		t.Fatalf("login failed: %d %s", resp.StatusCode, body)
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("parse login response: %v", err)
	}
	
	if token, ok := result["token"].(string); !ok || token == "" {
		t.Fatalf("token missing: %+v", result)
	}
	
	t.Logf("Login success, token: %s", result["token"])
}
