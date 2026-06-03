// ============================================================
// 16-native-stdlib.gs -- 原生标准库：文件、路径、系统信息与加密
// ============================================================
// 原生库由 Go 侧注册，脚本中通过 @std/... 路径加载。

let fs = require("@std/fs");
let path = require("@std/path");
let os = require("@std/os");
let process = require("@std/process");
let crypto = require("@std/crypto");

function main() {
  console.log("=== GoScript 原生标准库示例 ===");

  let root = path.join(os.tmpdir(), "goscript-native-stdlib-demo-" + crypto.randomUUID());
  fs.rmSync(root, { recursive: true, force: true });
  fs.mkdirSync(root, { recursive: true });

  let note = path.join(root, "note.txt");
  let digest = crypto.sha256("GoScript native stdlib");

  fs.writeTextSync(note, "digest=" + digest + os.eol);
  fs.appendFileSync(note, "cwd=" + process.cwd() + os.eol);

  let copy = path.join(root, "note.copy.txt");
  fs.copyFileSync(note, copy);

  let stat = fs.statSync(copy);
  let parsed = path.parse(copy);
  let text = fs.readTextSync(copy);

  console.log("临时目录:", root);
  console.log("复制文件:", parsed.base);
  console.log("是否文件:", stat.isFile(), "大小:", stat.size);
  console.log("SHA256:", digest);
  console.log("系统:", os.type(), os.arch);
  console.log("内容:");
  console.log(text);

  fs.rmSync(root, { recursive: true, force: true });
}

main();
