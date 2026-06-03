# 15-gs-agent

Small AI agent demo inspired by the `pi` agent architecture.

It demonstrates:

- provider + agent loop + tool registry composition
- Anthropic Messages API provider support
- workspace-safe file tools
- JSONL session recording
- absolute-style imports through `[imports]`

Run:

```powershell
go run ..\..\cmd\gs run
```

The default `agent.toml` uses a deterministic scripted provider, so it does not
need a network connection or API key.

Run with Anthropic:

```powershell
Copy-Item .\agent.local.example.toml .\agent.local.toml
# edit agent.local.toml with your provider endpoint, model, and API key
go run ..\..\cmd\gs run
```

DeepSeek's Anthropic-compatible endpoint can be configured like this:

```toml
[agent]
provider = "anthropic"
system = "You are a concise coding agent. Use tools when useful."
maxTurns = 4

[llm.anthropic]
apiKey = "sk-..."
baseUrl = "https://api.deepseek.com/anthropic"
model = "deepseek-v4-flash"
maxTokens = 1024
timeoutMs = 60000
```

`agent.local.toml` is ignored by git and is intended for local secrets.
