import { createTool } from "@agent/tools/registry";
import { workspacePath } from "@agent/tools/files";

let fs = require("@std/fs");
let path = require("@std/path");

function searchFile(root, relativePath, query, results, limit) {
  if (results.length >= limit) {
    return;
  }

  let fullPath = path.join(root, relativePath);
  let stat = fs.statSync(fullPath);
  if (!stat.isFile()) {
    return;
  }

  let text = fs.readFileSync(fullPath);
  if (!text.includes(query)) {
    return;
  }

  let lines = text.split("\n");
  for (let i = 0; i < lines.length; i = i + 1) {
    if (results.length >= limit) {
      return;
    }
    if (lines[i].includes(query)) {
      results.push({
        path: relativePath,
        line: i + 1,
        text: lines[i],
      });
    }
  }
}

function walk(root, relativePath, query, results, limit) {
  if (results.length >= limit) {
    return;
  }

  let fullPath = path.join(root, relativePath);
  let stat = fs.statSync(fullPath);
  if (stat.isFile()) {
    searchFile(root, relativePath, query, results, limit);
    return;
  }

  if (!stat.isDirectory()) {
    return;
  }

  let entries = fs.readdirSync(fullPath);
  for (let entry of entries) {
    if (entry === ".git" || entry === "node_modules" || entry === ".agent-smoke") {
      continue;
    }
    let child = entry;
    if (relativePath !== "." && relativePath !== "") {
      child = path.join(relativePath, entry);
    }
    walk(root, child, query, results, limit);
  }
}

export function createGrepTool(cwd) {
  return createTool(
    "grep",
    "Search UTF-8 text files in the workspace using plain string matching.",
    {
      type: "object",
      required: ["query"],
      additionalProperties: false,
      properties: {
        query: { type: "string", minLength: 1 },
        path: { type: "string" },
        limit: { type: "integer", minimum: 1, maximum: 100 },
      },
    },
    function(args) {
      let start = ".";
      if (args.path !== undefined) {
        start = args.path;
      }
      let limit = 20;
      if (args.limit !== undefined) {
        limit = args.limit;
      }

      let root = workspacePath(cwd, ".");
      let relativeStart = start;
      workspacePath(cwd, relativeStart);

      let results = [];
      walk(root, relativeStart, args.query, results, limit);
      return {
        query: args.query,
        matches: results,
      };
    }
  );
}
