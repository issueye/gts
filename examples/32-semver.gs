// 语义化版本示例
let semver = require("@std/semver");

console.log("=== 语义化版本示例 ===\n");

// 1. 解析版本
console.log("--- 解析版本 ---");
let v1 = semver.parse("1.2.3");
console.log("1.2.3:", JSON.stringify(v1));

let v2 = semver.parse("2.0.0-alpha.1+build.123");
console.log("2.0.0-alpha.1+build.123:", JSON.stringify(v2));

let v3 = semver.parse("v3.5.7");
console.log("v3.5.7 (带 v 前缀):", JSON.stringify(v3));

// 2. 验证版本
console.log("\n--- 验证版本 ---");
console.log("1.2.3 有效?", semver.valid("1.2.3"));
console.log("1.2 有效?", semver.valid("1.2"));
console.log("v1.2.3 有效?", semver.valid("v1.2.3"));
console.log("1.2.3-beta 有效?", semver.valid("1.2.3-beta"));

// 3. 版本比较
console.log("\n--- 版本比较 ---");
console.log("compare(1.2.3, 1.3.0):", semver.compare("1.2.3", "1.3.0"));
console.log("compare(1.2.3, 1.2.3):", semver.compare("1.2.3", "1.2.3"));
console.log("compare(1.3.0, 1.2.3):", semver.compare("1.3.0", "1.2.3"));

console.log("\ngt(1.3.0, 1.2.3):", semver.gt("1.3.0", "1.2.3"));
console.log("gte(1.2.3, 1.2.3):", semver.gte("1.2.3", "1.2.3"));
console.log("lt(1.2.0, 1.2.3):", semver.lt("1.2.0", "1.2.3"));
console.log("lte(1.2.3, 1.2.3):", semver.lte("1.2.3", "1.2.3"));
console.log("eq(1.2.3, 1.2.3):", semver.eq("1.2.3", "1.2.3"));
console.log("neq(1.2.3, 1.3.0):", semver.neq("1.2.3", "1.3.0"));

// 4. 预发布版本比较
console.log("\n--- 预发布版本比较 ---");
console.log("1.0.0-alpha < 1.0.0:", semver.lt("1.0.0-alpha", "1.0.0"));
console.log("1.0.0-alpha < 1.0.0-beta:", semver.lt("1.0.0-alpha", "1.0.0-beta"));
console.log("1.0.0-beta < 1.0.0:", semver.lt("1.0.0-beta", "1.0.0"));

// 5. 版本递增
console.log("\n--- 版本递增 ---");
let current = "1.2.3";
console.log("当前版本:", current);
console.log("  major:", semver.inc(current, "major"));
console.log("  minor:", semver.inc(current, "minor"));
console.log("  patch:", semver.inc(current, "patch"));
console.log("  prerelease:", semver.inc(current, "prerelease"));

// 6. 范围匹配 - ^ (兼容版本)
console.log("\n--- 范围匹配: ^ (兼容版本) ---");
let range1 = "^1.2.0";
console.log("范围:", range1);
console.log("  1.2.0 满足?", semver.satisfies("1.2.0", range1));
console.log("  1.2.5 满足?", semver.satisfies("1.2.5", range1));
console.log("  1.3.0 满足?", semver.satisfies("1.3.0", range1));
console.log("  2.0.0 满足?", semver.satisfies("2.0.0", range1));
console.log("  1.1.9 满足?", semver.satisfies("1.1.9", range1));

// 7. 范围匹配 - ~ (近似版本)
console.log("\n--- 范围匹配: ~ (近似版本) ---");
let range2 = "~1.2.0";
console.log("范围:", range2);
console.log("  1.2.0 满足?", semver.satisfies("1.2.0", range2));
console.log("  1.2.5 满足?", semver.satisfies("1.2.5", range2));
console.log("  1.3.0 满足?", semver.satisfies("1.3.0", range2));
console.log("  1.2.9 满足?", semver.satisfies("1.2.9", range2));

// 8. 范围匹配 - 表达式
console.log("\n--- 范围匹配: 表达式 ---");
let range3 = ">=1.2.0 <2.0.0";
console.log("范围:", range3);
console.log("  1.2.0 满足?", semver.satisfies("1.2.0", range3));
console.log("  1.5.0 满足?", semver.satisfies("1.5.0", range3));
console.log("  2.0.0 满足?", semver.satisfies("2.0.0", range3));
console.log("  1.1.9 满足?", semver.satisfies("1.1.9", range3));

// 9. 版本排序
console.log("\n--- 版本排序 ---");
let versions = ["1.3.0", "1.2.5", "2.0.0", "1.2.3", "1.10.0"];
console.log("原始:", versions.join(", "));

versions.sort((a, b) => semver.compare(a, b));
console.log("排序后:", versions.join(", "));

// 10. 实际应用 - 依赖检查
console.log("\n--- 实际应用: 依赖检查 ---");

let dependencies = {
  "express": "^4.17.0",
  "react": "~18.2.0",
  "lodash": ">=4.17.0 <5.0.0"
};

let installed = {
  "express": "4.18.2",
  "react": "18.2.5",
  "lodash": "4.17.21"
};

console.log("依赖检查:");
for (let pkg in dependencies) {
  let required = dependencies[pkg];
  let current = installed[pkg];
  let ok = semver.satisfies(current, required);
  let status = ok ? "✓" : "✗";
  console.log("  " + status + " " + pkg + ": " + current + " (需要 " + required + ")");
}

// 11. 找出最新版本
console.log("\n--- 找出最新版本 ---");
let availableVersions = ["1.2.3", "1.2.5", "1.3.0", "1.2.4", "2.0.0"];
console.log("可用版本:", availableVersions.join(", "));

let latest = availableVersions[0];
for (let i = 1; i < availableVersions.length; i++) {
  if (semver.gt(availableVersions[i], latest)) {
    latest = availableVersions[i];
  }
}
console.log("最新版本:", latest);

// 12. 版本发布流程
console.log("\n--- 版本发布流程 ---");
let projectVersion = "2.5.8";
console.log("当前版本:", projectVersion);
console.log("\n下个版本建议:");
console.log("  补丁版本 (bug 修复):", semver.inc(projectVersion, "patch"));
console.log("  次版本 (新功能):", semver.inc(projectVersion, "minor"));
console.log("  主版本 (破坏性变更):", semver.inc(projectVersion, "major"));
console.log("  预发布版本:", semver.inc(projectVersion, "prerelease"));

// 13. 版本兼容性检查
console.log("\n--- 版本兼容性检查 ---");
let minVersion = "1.5.0";
let userVersions = ["1.4.9", "1.5.0", "1.6.2", "2.0.0"];

console.log("最低要求版本:", minVersion);
console.log("用户版本兼容性:");
for (let i = 0; i < userVersions.length; i++) {
  let compatible = semver.gte(userVersions[i], minVersion);
  let status = compatible ? "✓ 兼容" : "✗ 不兼容";
  console.log("  " + userVersions[i] + ": " + status);
}

console.log("\n=== 所有测试完成 ===");
