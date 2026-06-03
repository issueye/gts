let http = require("@std/net/http/client");

function asTextContent(content) {
  if (typeof content === "string") {
    return content;
  }
  return JSON.stringify(content);
}

function toAnthropicMessages(messages) {
  let out = [];
  for (let message of messages) {
    if (message.kind === "tool_call") {
      out.push({
        role: "assistant",
        content: [
          {
            type: "tool_use",
            id: message.id,
            name: message.name,
            input: message.args,
          },
        ],
      });
      continue;
    }

    if (message.role === "tool") {
      out.push({
        role: "user",
        content: [
          {
            type: "tool_result",
            tool_use_id: message.id,
            content: asTextContent(message.content),
          },
        ],
      });
      continue;
    }

    if (message.role === "user" || message.role === "assistant") {
      out.push({
        role: message.role,
        content: asTextContent(message.content),
      });
    }
  }
  return out;
}

function toAnthropicTools(tools) {
  return tools.map(function(tool) {
    return {
      name: tool.name,
      description: tool.description,
      input_schema: tool.inputSchema,
    };
  });
}

function firstText(blocks) {
  let text = "";
  for (let block of blocks) {
    if (block.type === "text") {
      text = text + block.text;
    }
  }
  return text;
}

function firstToolUse(blocks) {
  for (let block of blocks) {
    if (block.type === "tool_use") {
      return block;
    }
  }
  return undefined;
}

export function createAnthropicProvider(options) {
  if (options === undefined) {
    options = {};
  }

  let apiKey = options.apiKey;
  if (apiKey === undefined || apiKey === "") {
    throw new ReferenceError("createAnthropicProvider requires options.apiKey");
  }

  let baseUrl = options.baseUrl;
  if (baseUrl === undefined) {
    baseUrl = "https://api.anthropic.com";
  }

  let model = options.model;
  if (model === undefined) {
    model = "claude-3-5-sonnet-latest";
  }

  let maxTokens = options.maxTokens;
  if (maxTokens === undefined) {
    maxTokens = 1024;
  }

  let system = options.system;
  if (system === undefined) {
    system = "You are a concise coding assistant.";
  }

  function next(messages, tools) {
    let body = {
      model: model,
      max_tokens: maxTokens,
      system: system,
      messages: toAnthropicMessages(messages),
      tools: toAnthropicTools(tools),
    };

    if (options.temperature !== undefined) {
      body.temperature = options.temperature;
    }

    let response = http.request({
      method: "POST",
      url: baseUrl + "/v1/messages",
      timeoutMs: options.timeoutMs || 60000,
      headers: {
        "content-type": "application/json",
        "x-api-key": apiKey,
        "anthropic-version": "2023-06-01",
      },
      body: JSON.stringify(body),
    });

    if (!response.ok) {
      throw new Error("Anthropic request failed: " + response.status + " " + response.body);
    }

    let payload = JSON.parse(response.body);
    let toolUse = firstToolUse(payload.content);
    if (toolUse !== undefined) {
      return {
        kind: "tool_call",
        id: toolUse.id,
        name: toolUse.name,
        args: toolUse.input,
      };
    }

    return {
      role: "assistant",
      content: firstText(payload.content),
    };
  }

  return {
    next: next,
  };
}
