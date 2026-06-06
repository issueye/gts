// ============================================================
// 23-native-tui.gs -- 原生 TUI 框架：状态机、布局和可选全屏运行
// ============================================================

let tui = require("@std/tui");
let process = require("@std/process");

function main() {
  let app = tui.createApp({
    init: function(size) {
      return {
        title: "GoScript TUI",
        count: 0,
        last: "ready",
        cols: size.cols,
      };
    },
    update: function(state, msg) {
      if (msg.type === "resize") {
        state.cols = msg.cols;
        state.last = "resize " + String(msg.cols) + "x" + String(msg.rows);
        return state;
      }
      if (msg.type === "text") {
        state.count = state.count + 1;
        state.last = "typed " + msg.text;
        return state;
      }
      if (msg.type === "key") {
        state.last = "key " + msg.key;
        if (msg.key === "ctrl+c") {
          return { state: state, quit: true };
        }
      }
      return state;
    },
    view: function(state, size) {
      let body = [
        tui.style(state.title, { bold: true, fg: "cyan" }),
        "",
        "count: " + String(state.count),
        "last:  " + state.last,
        "",
        "Press Ctrl+C to quit in --run mode.",
      ];
      let panel = tui.box(body, {
        title: "Demo",
        width: size.cols,
        padding: 1,
      });
      let footer = tui.statusBar({
        left: " @std/tui ",
        right: String(size.cols) + " cols ",
      }, size.cols);
      return tui.column(panel, footer);
    },
  });

  app.dispatch(tui.text("a"));
  app.dispatch(tui.resize(60, 12));

  if (process.argv.indexOf("--run") >= 0) {
    app.run({ tickMs: 1000 });
  } else {
    console.log(app.render({ cols: 60, rows: 12 }));
    console.log("");
    console.log("Run with --run for the interactive full-screen version.");
  }
}

main();
