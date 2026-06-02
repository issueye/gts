let fs = require("@std/fs");
let path = require("@std/path");

let root = path.join(".", ".agent-smoke");
let nested = path.join(root, "nested");
let file = path.join(nested, "message.txt");
let copy = path.join(root, "copy.txt");

fs.mkdirSync(nested, { recursive: true });
fs.writeFileSync(file, "agent std ok");
fs.copyFileSync(file, copy);
let tmpDir = fs.mkdtempSync(path.join(root, "tmp-"));

let stat = fs.statSync(file);
let kind = "not-file";
if (stat.isFile()) {
  kind = "file";
}

let slashKind = "slash-bad";
if (path.toSlash(path.fromSlash("agent/std/message.txt")) === "agent/std/message.txt") {
  slashKind = "slash-ok";
}

let matchKind = "match-bad";
if (path.matches("*.txt", path.basename(file))) {
  matchKind = "match-ok";
}

let parsed = path.parse(file);
let parseKind = "parse-bad";
if (parsed.name === "message" && path.basename(path.format(parsed)) === "message.txt") {
  parseKind = "parse-ok";
}

let typed = fs.readdirSync(root, { withFileTypes: true });
let entryKind = "entry-bad";
for (let i = 0; i < typed.length; i = i + 1) {
  if (typed[i].name === "nested" && typed[i].isDirectory()) {
    entryKind = "entry-ok";
  }
}

let realKind = "real-bad";
if (path.isAbs(fs.realpathSync(copy)) && fs.lstatSync(copy).isFile()) {
  realKind = "real-ok";
}

let globbed = fs.globSync(path.join(root, "*.txt"));
let globKind = "glob-bad";
if (globbed.length === 1) {
  globKind = "glob-one";
}
let tmpKind = "tmp-bad";
if (fs.existsSync(tmpDir)) {
  tmpKind = "tmp-ok";
}
println(path.basename(file) + ":" + fs.readFileSync(copy) + ":" + kind + ":" + slashKind + ":" + matchKind + ":" + parseKind + ":" + entryKind + ":" + realKind + ":" + globKind + ":" + tmpKind);

fs.rmSync(root, { recursive: true, force: true });
