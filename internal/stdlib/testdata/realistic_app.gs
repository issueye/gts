// TestRealisticWebApp is an end-to-end test of a realistic web application
// built with isolated mode + @std/shared: user registration, login, session
// management, and protected routes. Tests the full stack under real-world load.

let web = require("@std/web");
let shared = require("@std/shared");

let app = web.createApp({ concurrency: "isolated", poolSize: 8 });

// Shared state: users database and active sessions
let users = shared.map("app-users");       // username -> {password, created}
let sessions = shared.map("app-sessions"); // sessionId -> {username, expires}
let requestCount = shared.counter("app-requests");

// Middleware: log all requests
app.use(function(req, res, next) {
  requestCount.incr();
  return next();
});

// Middleware: extract session token from Authorization header
app.use(function(req, res, next) {
  let auth = req.headers["authorization"];
  if (auth && auth.indexOf("Bearer ") === 0) {
    let token = auth.substring(7);
    let sessionJson = sessions.get(token);
    if (sessionJson !== undefined) {
      let session = JSON.parse(sessionJson);
      let now = Date.now();
      if (session.expires > now) {
        req.user = session.username;
      }
    }
  }
  return next();
});

// Route: register new user
app.post("/register", function(req, res) {
  let data = JSON.parse(req.body);
  let username = data.username;
  let password = data.password;
  
  if (!username || !password) {
    res.status(400).json({ error: "username and password required" });
    return;
  }
  
  if (users.has(username)) {
    res.status(409).json({ error: "username already exists" });
    return;
  }
  
  // Store user as JSON string (for cross-VM compatibility)
  users.set(username, JSON.stringify({
    password: password,
    created: Date.now()
  }));
  
  res.status(201).json({ username: username, created: true });
});

// Route: login (create session)
app.post("/login", function(req, res) {
  let data = JSON.parse(req.body);
  let username = data.username;
  let password = data.password;
  
  let userJson = users.get(username);
  if (userJson === undefined) {
    res.status(401).json({ error: "invalid credentials" });
    return;
  }
  
  let user = JSON.parse(userJson);
  if (user.password !== password) {
    res.status(401).json({ error: "invalid credentials" });
    return;
  }
  
  // Generate session token (simple: username + timestamp)
  let sessionId = username + "-" + Date.now();
  let expires = Date.now() + 3600000; // 1 hour
  sessions.set(sessionId, JSON.stringify({
    username: username,
    expires: expires
  }));
  
  res.json({ token: sessionId, expires: expires });
});

// Route: logout (delete session)
app.post("/logout", function(req, res) {
  if (!req.user) {
    res.status(401).json({ error: "not authenticated" });
    return;
  }
  
  let auth = req.headers["authorization"];
  if (auth) {
    let token = auth.substring(7);
    sessions.delete(token);
  }
  
  res.json({ logged_out: true });
});

// Route: get current user profile (protected)
app.get("/me", function(req, res) {
  if (!req.user) {
    res.status(401).json({ error: "not authenticated" });
    return;
  }
  
  let userJson = users.get(req.user);
  if (userJson === undefined) {
    res.status(404).json({ error: "user not found" });
    return;
  }
  
  let user = JSON.parse(userJson);
  res.json({
    username: req.user,
    created: user.created
  });
});

// Route: stats (protected, admin only for demo purposes we allow all authenticated)
app.get("/stats", function(req, res) {
  if (!req.user) {
    res.status(401).json({ error: "not authenticated" });
    return;
  }
  
  res.json({
    total_requests: requestCount.get(),
    total_users: users.keys().length,
    active_sessions: sessions.keys().length
  });
});

let server = app.listen(0);
app;
