package stdlib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

// TestRealisticWebAppEndToEnd is a comprehensive functional test of a real-world
// web app: user registration, login, session management, protected routes, and
// concurrent multi-user workflows.
func TestRealisticWebAppEndToEnd(t *testing.T) {
	// Load the realistic app script
	scriptPath := filepath.Join("testdata", "realistic_app.gs")
	src, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read realistic_app.gs: %v", err)
	}

	app := evalWebIsolatedApp(t, string(src))
	server := httptest.NewServer(app)
	defer server.Close()

	// Helper: register a user
	registerUser := func(username, password string) (int, map[string]interface{}) {
		body, _ := json.Marshal(map[string]string{"username": username, "password": password})
		resp, err := http.Post(server.URL+"/register", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		return resp.StatusCode, result
	}

	// Helper: login a user
	loginUser := func(username, password string) (int, string) {
		body, _ := json.Marshal(map[string]string{"username": username, "password": password})
		resp, err := http.Post(server.URL+"/login", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		var result map[string]interface{}
		if err := json.Unmarshal(respBody, &result); err != nil {
			t.Logf("login response parse error: %v, body: %q", err, respBody)
			return resp.StatusCode, ""
		}
		token := ""
		if t, ok := result["token"].(string); ok {
			token = t
		}
		return resp.StatusCode, token
	}

	// Helper: get current user profile
	getMe := func(token string) (int, map[string]interface{}) {
		req, _ := http.NewRequest("GET", server.URL+"/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		return resp.StatusCode, result
	}

	// Helper: logout
	logout := func(token string) int {
		req, _ := http.NewRequest("POST", server.URL+"/logout", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}

	// Helper: get stats
	getStats := func(token string) (int, map[string]interface{}) {
		req, _ := http.NewRequest("GET", server.URL+"/stats", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		return resp.StatusCode, result
	}

	// Test 1: Register a user
	status, result := registerUser("alice", "secret123")
	if status != 201 {
		t.Fatalf("register alice: want 201, got %d: %v", status, result)
	}
	if result["username"] != "alice" || result["created"] != true {
		t.Fatalf("register response unexpected: %v", result)
	}

	// Test 2: Register duplicate user (should fail)
	status, result = registerUser("alice", "different")
	if status != 409 {
		t.Fatalf("register duplicate alice: want 409, got %d: %v", status, result)
	}

	// Test 3: Login with correct credentials
	status, token := loginUser("alice", "secret123")
	if status != 200 || token == "" {
		t.Fatalf("login alice: want 200 with token, got %d token=%q", status, token)
	}

	// Test 4: Login with wrong password
	status, _ = loginUser("alice", "wrong")
	if status != 401 {
		t.Fatalf("login alice wrong password: want 401, got %d", status)
	}

	// Test 5: Get profile with valid token
	status, profile := getMe(token)
	if status != 200 || profile["username"] != "alice" {
		t.Fatalf("GET /me: want 200 alice, got %d: %v", status, profile)
	}

	// Test 6: Get profile without token (should fail)
	status, _ = getMe("")
	if status != 401 {
		t.Fatalf("GET /me unauthenticated: want 401, got %d", status)
	}

	// Test 7: Check stats (authenticated)
	status, stats := getStats(token)
	if status != 200 {
		t.Fatalf("GET /stats: want 200, got %d: %v", status, stats)
	}
	totalUsers := int(stats["total_users"].(float64))
	if totalUsers < 1 {
		t.Fatalf("stats total_users = %d, want >= 1", totalUsers)
	}

	// Test 8: Logout
	status = logout(token)
	if status != 200 {
		t.Fatalf("logout: want 200, got %d", status)
	}

	// Test 9: After logout, token should be invalid
	status, _ = getMe(token)
	if status != 401 {
		t.Fatalf("GET /me after logout: want 401, got %d", status)
	}

	// Test 10: Concurrent multi-user registration and login
	const users = 30
	var wg sync.WaitGroup
	var failures int64
	tokens := make([]string, users)
	for i := 0; i < users; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			username := fmt.Sprintf("user%d", id)
			password := fmt.Sprintf("pass%d", id)

			// Register
			status, _ := registerUser(username, password)
			if status != 201 {
				atomic.AddInt64(&failures, 1)
				return
			}

			// Login
			status, tok := loginUser(username, password)
			if status != 200 || tok == "" {
				atomic.AddInt64(&failures, 1)
				return
			}
			tokens[id] = tok

			// Get profile
			status, profile := getMe(tok)
			if status != 200 || profile["username"] != username {
				atomic.AddInt64(&failures, 1)
				return
			}
		}(i)
	}
	wg.Wait()

	if f := atomic.LoadInt64(&failures); f > 0 {
		t.Fatalf("%d/%d concurrent user workflows failed", f, users)
	}

	// Test 11: Verify final stats reflect all users
	// Re-login alice to get a valid token
	_, aliceToken := loginUser("alice", "secret123")
	status, finalStats := getStats(aliceToken)
	if status != 200 {
		t.Fatalf("final stats: want 200, got %d", status)
	}
	finalUsers := int(finalStats["total_users"].(float64))
	// alice + 30 concurrent users = 31 total
	if finalUsers != users+1 {
		t.Fatalf("final total_users = %d, want %d", finalUsers, users+1)
	}

	t.Logf("Realistic app test: 31 users, concurrent registration/login/profile, all passed")
}

// TestRealisticWebAppUnderLoad is a load test: simulate realistic mixed traffic
// (registration, login, profile reads, stats) under sustained concurrent load.
func TestRealisticWebAppUnderLoad(t *testing.T) {
	scriptPath := filepath.Join("testdata", "realistic_app.gs")
	src, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read realistic_app.gs: %v", err)
	}

	app := evalWebIsolatedApp(t, string(src))
	server := httptest.NewServer(app)
	defer server.Close()

	// Pre-register 10 users
	tokens := make([]string, 10)
	for i := 0; i < 10; i++ {
		username := fmt.Sprintf("load%d", i)
		password := "loadpass"
		body, _ := json.Marshal(map[string]string{"username": username, "password": password})
		resp, _ := http.Post(server.URL+"/register", "application/json", bytes.NewReader(body))
		resp.Body.Close()

		// Login to get token
		resp, _ = http.Post(server.URL+"/login", "application/json", bytes.NewReader(body))
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		tokens[i] = result["token"].(string)
	}

	// Generate load: 300 requests (mix of /me, /stats, /login)
	const requests = 300
	var wg sync.WaitGroup
	var failures int64
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			token := tokens[id%len(tokens)]
			route := id % 3

			var resp *http.Response
			var err error
			switch route {
			case 0: // GET /me
				req, _ := http.NewRequest("GET", server.URL+"/me", nil)
				req.Header.Set("Authorization", "Bearer "+token)
				resp, err = http.DefaultClient.Do(req)
			case 1: // GET /stats
				req, _ := http.NewRequest("GET", server.URL+"/stats", nil)
				req.Header.Set("Authorization", "Bearer "+token)
				resp, err = http.DefaultClient.Do(req)
			case 2: // POST /login (re-login)
				username := fmt.Sprintf("load%d", id%len(tokens))
				body, _ := json.Marshal(map[string]string{"username": username, "password": "loadpass"})
				resp, err = http.Post(server.URL+"/login", "application/json", bytes.NewReader(body))
			}

			if err != nil {
				atomic.AddInt64(&failures, 1)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				atomic.AddInt64(&failures, 1)
			}
			io.Copy(io.Discard, resp.Body)
		}(i)
	}
	wg.Wait()

	if f := atomic.LoadInt64(&failures); f > 0 {
		t.Fatalf("%d/%d requests failed under load", f, requests)
	}

	t.Logf("Load test: %d requests (mixed routes), all passed", requests)
}
