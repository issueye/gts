import { createCodingAgent } from "@agent/core/kit";
import { createScriptedProvider } from "@agent/llm/fake";
import { createAnthropicProvider } from "@agent/llm/anthropic";
import { createWorkspaceTools } from "@/tools/workspace";

let fs = require("@std/fs");
let path = require("@std/path");
let process = require("@std/process");
let toml = require("@std/toml");

function readConfig(root) {
  let localFile = path.join(root, "agent.local.toml");
  if (fs.existsSync(localFile)) {
    return toml.readFileSync(localFile);
  }

  let configFile = path.join(root, "agent.toml");
  if (fs.existsSync(configFile)) {
    return toml.readFileSync(configFile);
  }

  return {
    agent: {
      provider: "fake",
      system: "You are a concise coding agent. Use tools when useful.",
      maxTurns: 4,
    },
  };
}

function agentConfig(config) {
  if (config.agent === undefined) {
    return {
      provider: "fake",
      system: "You are a concise coding agent. Use tools when useful.",
      maxTurns: 4,
    };
  }
  return config.agent;
}

function anthropicConfig(config) {
  if (config.llm === undefined || config.llm.anthropic === undefined) {
    throw new ReferenceError("agent provider is anthropic, but [llm.anthropic] is missing");
  }
  return config.llm.anthropic;
}

function createProvider(config) {
  let agent = agentConfig(config);
  let providerName = agent.provider;
  if (providerName === undefined) {
    providerName = "fake";
  }

  if (providerName === "anthropic") {
    let anthropic = anthropicConfig(config);
    return createAnthropicProvider({
      apiKey: anthropic.apiKey,
      baseUrl: anthropic.baseUrl,
      model: anthropic.model,
      maxTokens: anthropic.maxTokens,
      timeoutMs: anthropic.timeoutMs,
      temperature: anthropic.temperature,
      system: agent.system,
    });
  }

  return createScriptedProvider([
    {
      kind: "tool_call",
      name: "read_task",
      args: { path: "task.txt" },
    },
    {
      role: "assistant",
      content: "Next action: inspect workspace files.",
    },
  ]);
}

export function runDemo() {
  let root = process.cwd();
  let config = readConfig(root);
  let agent = agentConfig(config);
  let workspace = path.join(root, "workspace");
  let sessionFile = path.join(root, ".agent", "session.jsonl");

  if (fs.existsSync(sessionFile)) {
    fs.unlinkSync(sessionFile);
  }

  let provider = createProvider(config);
  let maxTurns = agent.maxTurns;
  if (maxTurns === undefined) {
    maxTurns = 4;
  }

  let kit = createCodingAgent({
    cwd: root,
    includeCodingTools: false,
    provider: provider,
    tools: createWorkspaceTools(workspace),
    sessionFile: sessionFile,
    maxTurns: maxTurns,
  });

  let answer = kit.agent.run("Read the task file and decide the next action.");
  let records = kit.session.readAll();

  return {
    answer: answer.content,
    events: records.length,
    sessionFile: sessionFile,
  };
}
