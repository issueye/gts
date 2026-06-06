let terminal = require("@std/terminal");

let tty = terminal.isTTY("stdout");
let size = terminal.size();
let caps = terminal.capabilities();
let session = terminal.start({ raw: false, resizeDebounceMs: 50 });
let sessionSize = session.size();
let writeCount = session.write("");
let clearCount = session.clear({ screen: false, scrollback: false });
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
let capsKind = "bad";
let renderKind = "bad";
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
if (caps.clearScrollback === true && caps.alternateScreen === true && caps.resizeEvents === true) {
  capsKind = "ok";
}
if (clearCount === 0 && session.renderFrame !== undefined && session.redraw !== undefined) {
  renderKind = "ok";
}
if (offResizeCount === 1) {
  listenerKind = "ok";
}

println("terminal:" + kind);
println("terminal-start:" + startKind);
println("terminal-caps:" + capsKind);
println("terminal-render:" + renderKind);
println("terminal-listener:" + listenerKind);
