package stdlib

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestDateNowInIsolatedMode verifies that Date.now() works in isolated handlers.
func TestDateNowInIsolatedMode(t *testing.T) {
	src := `
let web = require("@std/web");
let app = web.createApp({ concurrency: "isolated", poolSize: 2 });

app.get("/now", function(req, res) {
  let ts = Date.now();
  res.send("ts=" + ts);
});

app.get("/concat", function(req, res) {
  let token = "prefix-" + Date.now();
  res.send(token);
});

let server = app.listen(0);
app;
`
	app := evalWebIsolatedApp(t, src)
	server := httptest.NewServer(app)
	defer server.Close()

	// Test Date.now() alone
	resp, err := http.Get(server.URL + "/now")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	t.Logf("GET /now: %d %s", resp.StatusCode, body)
	if resp.StatusCode != 200 {
		t.Fatalf("Date.now() failed: %d %s", resp.StatusCode, body)
	}

	// Test string concatenation with Date.now()
	resp, err = http.Get(server.URL + "/concat")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	t.Logf("GET /concat: %d %s", resp.StatusCode, body)
	if resp.StatusCode != 200 {
		t.Fatalf("concat with Date.now() failed: %d %s", resp.StatusCode, body)
	}
}
