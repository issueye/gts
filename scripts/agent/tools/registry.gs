let schema = require("@std/schema");

export function createTool(name, description, inputSchema, run) {
  return {
    name: name,
    description: description,
    inputSchema: inputSchema,
    run: run,
  };
}

export function createRegistry() {
  let tools = [];

  function register(tool) {
    tools.push(tool);
    return tool;
  }

  function registerAll(nextTools) {
    for (let tool of nextTools) {
      register(tool);
    }
    return tools.length;
  }

  function list() {
    return tools.map(function(tool) {
      return {
        name: tool.name,
        description: tool.description,
        inputSchema: tool.inputSchema,
      };
    });
  }

  function get(name) {
    for (let i = 0; i < tools.length; i = i + 1) {
      if (tools[i].name === name) {
        return tools[i];
      }
    }
    return undefined;
  }

  function call(name, args) {
    let tool = get(name);
    if (tool === undefined) {
      throw new ReferenceError("unknown tool: " + name);
    }

    let checked = schema.validate(tool.inputSchema, args);
    if (!checked.valid) {
      throw new TypeError("invalid args for " + name + ": " + checked.errors.join("; "));
    }

    return tool.run(args);
  }

  function safeCall(name, args) {
    try {
      return {
        ok: true,
        name: name,
        result: call(name, args),
      };
    } catch (err) {
      return {
        ok: false,
        name: name,
        error: String(err),
      };
    }
  }

  return {
    register: register,
    registerAll: registerAll,
    list: list,
    get: get,
    call: call,
    safeCall: safeCall,
  };
}
