// GTS 标准库功能验证
console.log("GTS 标准库验证开始...");

let passed = 0;
let failed = 0;

// P0 模块
console.log("\nP0 - 工程化基础:");
try { require("@std/test"); console.log("✅ test"); passed++; } catch(e) { console.log("❌ test"); failed++; }
try { require("@std/env"); console.log("✅ env"); passed++; } catch(e) { console.log("❌ env"); failed++; }
try { require("@std/json"); console.log("✅ json"); passed++; } catch(e) { console.log("❌ json"); failed++; }
try { require("@std/validation"); console.log("✅ validation"); passed++; } catch(e) { console.log("❌ validation"); failed++; }

// P1 模块
console.log("\nP1 - 高频工具:");
try { require("@std/collections"); console.log("✅ collections"); passed++; } catch(e) { console.log("❌ collections"); failed++; }
try { require("@std/random"); console.log("✅ random"); passed++; } catch(e) { console.log("❌ random"); failed++; }
try { require("@std/color"); console.log("✅ color"); passed++; } catch(e) { console.log("❌ color"); failed++; }
try { require("@std/semver"); console.log("✅ semver"); passed++; } catch(e) { console.log("❌ semver"); failed++; }
try { require("@std/cache"); console.log("✅ cache"); passed++; } catch(e) { console.log("❌ cache"); failed++; }

// P2 模块
console.log("\nP2 - 特定场景:");
try { require("@std/retry"); console.log("✅ retry"); passed++; } catch(e) { console.log("❌ retry"); failed++; }
try { require("@std/rate-limit"); console.log("✅ rate-limit"); passed++; } catch(e) { console.log("❌ rate-limit"); failed++; }
try { require("@std/glob"); console.log("✅ glob"); passed++; } catch(e) { console.log("❌ glob"); failed++; }
try { require("@std/diff"); console.log("✅ diff"); passed++; } catch(e) { console.log("❌ diff"); failed++; }
try { require("@std/regexp"); console.log("✅ regexp"); passed++; } catch(e) { console.log("❌ regexp"); failed++; }
try { require("@std/jwt"); console.log("✅ jwt"); passed++; } catch(e) { console.log("❌ jwt"); failed++; }
try { require("@std/watch"); console.log("✅ watch"); passed++; } catch(e) { console.log("❌ watch"); failed++; }
try { require("@std/compression"); console.log("✅ compression"); passed++; } catch(e) { console.log("❌ compression"); failed++; }
try { require("@std/pdf"); console.log("✅ pdf"); passed++; } catch(e) { console.log("❌ pdf"); failed++; }
try { require("@std/image"); console.log("✅ image"); passed++; } catch(e) { console.log("❌ image"); failed++; }
try { require("@std/prometheus"); console.log("✅ prometheus"); passed++; } catch(e) { console.log("❌ prometheus"); failed++; }

console.log("\n总计: " + (passed + failed) + " 个模块");
console.log("通过: " + passed);
console.log("失败: " + failed);
console.log("成功率: " + Math.floor((passed / (passed + failed)) * 100) + "%");
