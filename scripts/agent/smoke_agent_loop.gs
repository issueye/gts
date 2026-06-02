import { createRegistry } from "@agent/tools/registry";
import { createCodingTools } from "@agent/tools/coding";
import { createJSONLSession } from "@agent/session/jsonl";
import { createAgent } from "@agent/core/agent";
import { createFakeToolProvider } from "@agent/llm/fake";

let process = require("@std/process");
let fs = require("@std/fs");

let registry = createRegistry();
for (let tool of createCodingTools(process.cwd())) {
  registry.register(tool);
}

registry.call("write_file", {
  path: ".agent-smoke/loop.txt",
  content: "hello agent loop",
});

let session = createJSONLSession(".agent-smoke/loop-session.jsonl");
let provider = createFakeToolProvider(
  "read_file",
  { path: ".agent-smoke/loop.txt" },
  "read_file completed"
);

let agent = createAgent({
  provider: provider,
  registry: registry,
  session: session,
});

let answer = agent.run("please read loop file");
let records = session.readAll();

let answerKind = "answer-bad";
if (answer.content === "read_file completed") {
  answerKind = "answer-ok";
}

let recordKind = "records-bad";
if (records.length === 4) {
  recordKind = "records-4";
}

fs.unlinkSync(".agent-smoke/loop.txt");
fs.unlinkSync(".agent-smoke/loop-session.jsonl");

println(answerKind + ":" + recordKind);
