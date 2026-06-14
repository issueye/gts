package stdlib

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestRealisticAppDebug is a minimal test to debug the realistic app.
func TestRealisticAppDebug(t *testing.T) {
	scriptPath := filepath.Join("testdata", "realistic_app.gs")
	src, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read realistic_app.gs: %v", err)
	}

	app := evalWebIsolatedApp(t, string(src))
	server := httptest.NewServer(app)
	defer server.Close()

	// Register a user
	regBody, _ := json.Marshal(map[string]string{"username": "debug", "password": "test"})
	resp, err := http.Post(server.URL+"/register", "application/json", bytes.NewReader(regBody))
	if err != nil {
		t.Fatal(err)
	}
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	t.Logf("POST /register: %d %s", resp.StatusCode, respBody)

	// Login
	loginBody, _ := json.Marshal(map[string]string{"username": "debug", "password": "test"})
	resp, err = http.Post(server.URL+"/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatal(err)
	}
	respBody, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	t.Logf("POST /login: %d %s", resp.StatusCode, respBody)

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}
	t.Logf("Login result: %+v", result)
	
	if token, ok := result["token"].(string); ok && token != "" {
		t.Logf("Token: %s", token)
	} else {
		t.Fatalf("token missing or empty in response")
	}
}
