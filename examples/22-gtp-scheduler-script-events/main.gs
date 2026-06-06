const fs = require("@std/fs");
const scheduler = require("@plugin/scheduler");

function main() {
  scheduler.once("trigger", function(event) {
    let task = event.data;
    let payload = task.payload;
    let text = "event=" + event.event +
      ",task=" + task.id +
      ",name=" + task.name +
      ",fired=" + String(task.fired) +
      ",kind=" + payload.kind +
      ",message=" + payload.message;
    fs.writeFileSync("script-event-result.txt", text);
    println("script listener handled: " + payload.message);
  });

  scheduler.schedule({
    name: "script-listener-demo",
    delayMs: 100,
    payload: {
      kind: "writeFile",
      message: "handled in script"
    }
  });
}
