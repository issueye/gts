let terminal = require("@std/terminal");

let tty = terminal.isTTY("stdout");
let size = terminal.size();
let session = terminal.start({ raw: false });
let sessionSize = session.size();
let writeCount = session.write("");
session.stop();
function resizeListener(size) {
  return size.cols;
}
let resizeHandle = terminal.onResize(resizeListener);
let offResizeCount = terminal.offResize(resizeListener);
resizeHandle.stop();

let kind = "bad";
let ttyKind = "bad";
let startKind = "bad";
let listenerKind = "bad";
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
if (sessionSize.cols > 0 && sessionSize.rows > 0 && writeCount === 0) {
  startKind = "ok";
}
if (offResizeCount === 1) {
  listenerKind = "ok";
}

println("terminal:" + kind);
println("terminal-start:" + startKind);
println("terminal-listener:" + listenerKind);
