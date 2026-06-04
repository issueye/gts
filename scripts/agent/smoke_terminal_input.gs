let fs = require("@std/fs");
let path = require("@std/path");
let process = require("@std/process");
let pty = require("@std/pty");

let root = process.cwd();
let childPath = path.join(root, ".agent", "terminal-input-smoke-child.gs");
fs.mkdirSync(path.dirname(childPath), { recursive: true });
fs.writeFileSync(childPath, `
let terminal = require("@std/terminal");
let timers = require("@std/timers");
let seen = "";
let session = null;
function onInput(data) {
  seen = seen + data;
  if (seen.includes("xy")) {
    println("terminal-input:" + seen);
    session.stop();
  }
}
session = terminal.start({ raw: true, onInput: onInput });
timers.setTimeout(function() {
  println("terminal-input-timeout:" + seen);
  session.stop();
}, 1000);
`);

let cmdName = process.getenv("GOSCRIPT_GO", "go");
let cmdArgs = ["run", "./cmd/gs", childPath];

let term = pty.spawn(cmdName, cmdArgs, { cwd: root, cols: 80, rows: 24, timeoutMs: 5000 });
term.write("xy");
let output = "";
for (let i = 0; i < 20; i = i + 1) {
  let chunk = term.readText(4096, 500);
  if (chunk !== null) {
    output = output + chunk;
    if (output.includes("terminal-input:xy") || output.includes("terminal-input-timeout:")) {
      i = 20;
    }
  }
}
let result = term.wait();
term.close();

let kind = "bad";
if (output.includes("terminal-input:xy") && result.success) {
  kind = "ok";
}
println("terminal-input:" + kind);
