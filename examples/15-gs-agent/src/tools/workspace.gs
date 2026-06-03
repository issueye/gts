import { createTool } from "@agent/tools/registry";

let fs = require("@std/fs");
let path = require("@std/path");

function workspacePath(root, requested) {
  let base = path.resolve(root);
  let target = path.resolve(path.join(base, requested));
  let prefix = base + path.sep;

  if (target !== base && !target.startsWith(prefix)) {
    throw new RangeError("path is outside workspace: " + requested);
  }

  return target;
}

export function createReadTaskTool(root) {
  return createTool(
    "read_task",
    "Read a task file from the agent workspace.",
    {
      type: "object",
      required: ["path"],
      additionalProperties: false,
      properties: {
        path: { type: "string", minLength: 1 },
      },
    },
    function(args) {
      let target = workspacePath(root, args.path);
      return {
        path: args.path,
        content: fs.readFileSync(target),
      };
    }
  );
}

export function createWorkspaceTools(root) {
  return [
    createReadTaskTool(root),
  ];
}
