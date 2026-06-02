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

  return {
    register: register,
    list: list,
    get: get,
    call: call,
  };
}
