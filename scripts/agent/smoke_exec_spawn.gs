let exec = require("@std/exec");
let process = require("@std/process");

let cmdName = "bash";
let cmdArgs = ["-lc", "while IFS= read -r line; do echo got:$line; done"];
if (process.env.OS === "Windows_NT") {
  cmdName = "powershell";
  cmdArgs = ["-NoProfile", "-Command", "$input | ForEach-Object { 'got:' + $_ }"];
}

let child = exec.spawn(cmdName, cmdArgs);
child.writeln("one");
child.stdin.writeln("two");
child.closeStdin();

let first = child.stdout.readLine();
let second = child.stdout.readLine();
let result = child.wait();

let kind = "bad";
if (first === "got:one" && second === "got:two" && result.success) {
  kind = "ok";
}

println("exec-spawn:" + kind);
