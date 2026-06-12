// GTS 20 个模块加载测试
console.log("开始验证 20 个模块...\n");

console.log("P0 模块:");
let test = require("@std/test");
console.log("1. test ✅");

let env = require("@std/env");
console.log("2. env ✅");

let json = require("@std/json");
console.log("3. json ✅");

let validation = require("@std/validation");
console.log("4. validation ✅");

console.log("\nP1 模块:");
let collections = require("@std/collections");
console.log("5. collections ✅");

let random = require("@std/random");
console.log("6. random ✅");

let color = require("@std/color");
console.log("7. color ✅");

let semver = require("@std/semver");
console.log("8. semver ✅");

let cache = require("@std/cache");
console.log("9. cache ✅");

console.log("\nP2 模块:");
let retry = require("@std/retry");
console.log("10. retry ✅");

let rateLimit = require("@std/rate-limit");
console.log("11. rate-limit ✅");

let glob = require("@std/glob");
console.log("12. glob ✅");

let diff = require("@std/diff");
console.log("13. diff ✅");

let regexp = require("@std/regexp");
console.log("14. regexp ✅");

let jwt = require("@std/jwt");
console.log("15. jwt ✅");

let watch = require("@std/watch");
console.log("16. watch ✅");

let compression = require("@std/compression");
console.log("17. compression ✅");

let pdf = require("@std/pdf");
console.log("18. pdf ✅");

let image = require("@std/image");
console.log("19. image ✅");

let prometheus = require("@std/prometheus");
console.log("20. prometheus ✅");

console.log("\n所有 20 个模块加载成功！✅");
