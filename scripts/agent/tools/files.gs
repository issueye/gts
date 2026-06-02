import { createTool } from "@agent/tools/registry";

let fs = require("@std/fs");
let path = require("@std/path");

export function workspacePath(cwd, requested) {
  let root = path.resolve(cwd);
  let target = path.resolve(path.join(root, requested));
  let prefix = root + path.sep;

  if (target !== root && !target.startsWith(prefix)) {
    throw new RangeError("path is outside workspace: " + requested);
  }

  return target;
}

export function createReadFileTool(cwd) {
  return createTool(
    "read_file",
    "Read a UTF-8 text file from the workspace.",
    {
      type: "object",
      required: ["path"],
      additionalProperties: false,
      properties: {
        path: { type: "string", minLength: 1 },
      },
    },
    function(args) {
      let target = workspacePath(cwd, args.path);
      return {
        path: args.path,
        content: fs.readFileSync(target),
      };
    }
  );
}

export function createWriteFileTool(cwd) {
  return createTool(
    "write_file",
    "Write a UTF-8 text file inside the workspace.",
    {
      type: "object",
      required: ["path", "content"],
      additionalProperties: false,
      properties: {
        path: { type: "string", minLength: 1 },
        content: { type: "string" },
      },
    },
    function(args) {
      let target = workspacePath(cwd, args.path);
      fs.writeFileSync(target, args.content);
      return {
        path: args.path,
        bytes: args.content.length,
      };
    }
  );
}

export function createListDirTool(cwd) {
  return createTool(
    "list_dir",
    "List directory entries inside the workspace.",
    {
      type: "object",
      required: ["path"],
      additionalProperties: false,
      properties: {
        path: { type: "string" },
      },
    },
    function(args) {
      let target = workspacePath(cwd, args.path);
      return {
        path: args.path,
        entries: fs.readdirSync(target),
      };
    }
  );
}

export function createFileTools(cwd) {
  return [
    createReadFileTool(cwd),
    createWriteFileTool(cwd),
    createListDirTool(cwd),
  ];
}
