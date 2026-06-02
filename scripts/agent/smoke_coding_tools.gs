import { createRegistry } from "@agent/tools/registry";
import { createCodingTools } from "@agent/tools/coding";
import { createJSONLSession } from "@agent/session/jsonl";

let process = require("@std/process");
let fs = require("@std/fs");

let registry = createRegistry();
let tools = createCodingTools(process.cwd());

for (let tool of tools) {
  registry.register(tool);
}

let session = createJSONLSession(".agent-smoke/session.jsonl");
session.append("user", { text: "run coding tools smoke" });

registry.call("write_file", {
  path: ".agent-smoke/search.txt",
  content: "alpha\nneedle here\nomega",
});

let grep = registry.call("grep", {
  query: "needle",
  path: ".agent-smoke",
  limit: 5,
});

let bash = registry.call("bash", {
  command: "Write-Output agent-bash-ok",
});

session.append("tool", {
  grepMatches: grep.matches.length,
  bashExitCode: bash.exitCode,
});

let records = session.readAll();
let grepKind = "no-match";
if (grep.matches.length === 1) {
  grepKind = "grep-ok";
}

let bashKind = "bash-bad";
if (bash.success) {
  bashKind = "bash-ok";
}

let recordKind = "records-bad";
if (records.length === 2) {
  recordKind = "records-2";
}

fs.unlinkSync(".agent-smoke/search.txt");
fs.unlinkSync(".agent-smoke/session.jsonl");

println(grepKind + ":" + bashKind + ":" + recordKind);
