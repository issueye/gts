function main() {
  const scheduler = require("@plugin/scheduler");
  let tasks = scheduler.list();
  println("GTP scheduler plugin auto-started: " + String(tasks.length) + " tasks");
}
