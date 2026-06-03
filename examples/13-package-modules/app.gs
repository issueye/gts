// ============================================================
// 13-package-modules —— 包模块声明、导出与引用
// ============================================================

import { label } from "tools";
import { decorate } from "tools/format";

let tools = require("tools");

let message = decorate(label("package modules"));
println(message);

// require("tools") 与 import "tools" 会命中同一个包模块缓存。
println("loaded: " + tools.name);

message;
