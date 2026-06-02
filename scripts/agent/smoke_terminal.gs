let terminal = require("@std/terminal");

let tty = terminal.isTTY("stdout");
let size = terminal.size();

let kind = "bad";
let ttyKind = "bad";
if (tty === true) {
  ttyKind = "ok";
}
if (tty === false) {
  ttyKind = "ok";
}
if (ttyKind === "ok") {
  if (size.cols > 0) {
    if (size.rows > 0) {
      kind = "ok";
    }
  }
}

println("terminal:" + kind);
