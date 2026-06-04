let pty = require("@std/pty");
let process = require("@std/process");

let cmdName = "bash";
let cmdArgs = ["-lc", "echo got:pty"];
if (process.env.OS === "Windows_NT") {
  cmdName = "C:\\WINDOWS\\System32\\WindowsPowerShell\\v1.0\\powershell.exe";
  cmdArgs = ["-NoProfile", "-Command", "Write-Output 'got:pty'"];
}

let term = pty.spawn(cmdName, cmdArgs, { cols: 80, rows: 24 });
let text = "";
for (let i = 0; i < 8; i = i + 1) {
  let chunk = term.readText(4096, 500);
  if (chunk === null) {
    // Keep waiting: Windows ConPTY can emit setup sequences before command output.
  } else {
    text = text + chunk;
    if (text.includes("got:pty")) {
      i = 8;
    }
  }
}
term.resize(100, 30);
let result = term.wait();
term.close();

let kind = "bad";
if (text.includes("got:pty") && result.success) {
  kind = "ok";
}

println("pty:" + kind);
