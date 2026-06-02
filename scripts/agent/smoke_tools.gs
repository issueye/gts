import { createRegistry } from "@agent/tools/registry";
import { createFileTools } from "@agent/tools/files";

let process = require("@std/process");

let registry = createRegistry();
let tools = createFileTools(process.cwd());

for (let tool of tools) {
  registry.register(tool);
}

registry.call("write_file", { path: ".agent-tool-smoke.txt", content: "tool ok" });
let read = registry.call("read_file", { path: ".agent-tool-smoke.txt" });
let listed = registry.call("list_dir", { path: "." });

let found = "missing";
for (let entry of listed.entries) {
  if (entry === ".agent-tool-smoke.txt") {
    found = "found";
  }
}

let fs = require("@std/fs");
fs.unlinkSync(".agent-tool-smoke.txt");

let count = "tools-unknown";
if (registry.list().length === 3) {
  count = "tools-3";
}

println(read.content + ":" + found + ":" + count);
