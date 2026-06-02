let fs = require("@std/fs");
let path = require("@std/path");

let root = path.join(".", ".agent-smoke");
let file = path.join(root, "message.txt");

fs.mkdirSync(root, { recursive: true });
fs.writeFileSync(file, "agent std ok");

let stat = fs.statSync(file);
let kind = "not-file";
if (stat.isFile()) {
  kind = "file";
}
println(path.basename(file) + ":" + fs.readFileSync(file) + ":" + kind);

fs.unlinkSync(file);
