// 打包辅助脚本 - 将标准库复制到外部目录
// Usage: gs scripts/copy-stdlib.gs [output_dir]

let fs = require("@std/fs");
let path = require("@std/path");

let outputDir = "stdlib";
if (ARGV.length > 2) {
  outputDir = ARGV[2];
}

console.log("准备复制标准库到: " + outputDir);

// 创建输出目录
if (!fs.existsSync(outputDir)) {
  fs.mkdirSync(outputDir, { recursive: true });
}

// 标准库模块列表
let modules = [
  "test", "env", "json", "validation",
  "collections", "random", "color", "semver", "cache",
  "retry", "rate-limit", "glob", "diff", "regexp",
  "jwt", "watch", "compression", "pdf", "image", "prometheus"
];

let sourceDir = "internal/stdlib";
let copied = 0;

for (let mod of modules) {
  let sourceFile = path.join(sourceDir, mod.replace("-", "_") + ".go");

  if (fs.existsSync(sourceFile)) {
    let targetFile = path.join(outputDir, mod + ".go");
    fs.copyFileSync(sourceFile, targetFile);
    console.log("✅ " + mod);
    copied++;
  } else {
    console.log("⚠️  " + mod + " (源文件不存在: " + sourceFile + ")");
  }
}

console.log("\n复制完成: " + copied + "/" + modules.length + " 个模块");
console.log("输出目录: " + path.resolve(outputDir));
