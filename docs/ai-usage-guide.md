# GoScript AI 精简使用指南

> 给 AI 生成 `.gs` 脚本使用的紧凑上下文。详细规范查：
> [`language-spec.md`](language-spec.md)、[`grammar.ebnf`](grammar.ebnf)、[`builtins.md`](builtins.md)、[`async-model.md`](async-model.md)。

## 1. 核心判断

GoScript 是 JS 风格动态脚本语言，但语义更严格：

- 文件后缀：`.gs`。
- 入口：`main.gs` 或 `gs run` 会自动调用顶层 `main()`；普通脚本可手动 `main();`。
- 禁止 `==` / `!=`，只用 `===` / `!==`。
- 禁止 `switch/case/default`，用 `match`。
- 禁止 `eval`、`with`、`arguments`。
- 未声明赋值报错；先 `let` / `const`。
- `+` 不做混合类型拼接：`"id=" + 1` 会报错，用 `` `id=${1}` `` 或 `"id=" + String(1)`。
- `sort()` 必须传比较函数：`arr.sort((a, b) => a - b)`。
- 系统能力优先用 `require("@std/...")` 原生库。

最小脚本：

```javascript
let process = require("@std/process");

function main() {
  console.log("cwd:", process.cwd());
}

main();
```

## 2. 运行

```bash
go run ./cmd/gs main.gs
go run ./cmd/gs --timeout 30s main.gs
go run ./cmd/gs --timeout 0 main.gs
go run ./cmd/gs --workers 1 main.gs
go run ./cmd/gs --check-types main.gs
```

项目：

```toml
[project]
name = "demo"
version = "0.1.0"
entry = "main.gs"
```

```bash
go run ./cmd/gs run
```

其它命令：`init [dir]`、`pack [dir] [out.gspkg]`、`dist [dir] [out]`、`bundle <entry.gs> [out.gs]`。

## 3. 语法速查

```javascript
let x = 1;
const name = "Ada";

function add(a: number, b: number): number {
  return a + b;
}

let twice = x => x * 2;

if (x > 0) {
  println("positive");
} else {
  println("other");
}

for (let i = 0; i < 3; i++) {}
for (let k in obj) {}   // key / index
for (let v of arr) {}   // value
while (x < 10) {}

let label = match status {
  200 => "OK",
  404 => "Not Found",
  500..599 => "Server Error",
  _ => "Unknown",
};

try {
  risky();
} catch (e) {
  console.error(e.name + ": " + e.message);
} finally {
  cleanup();
}
```

类型注解是可选的；启用 `--check-types` 后做基础运行时检查：

```javascript
let n: number = 1;
let xs: number[] = [1, 2];
let maybe: string | null = null;
let user: { name: string, age: number } = { name: "Ada", age: 37 };
```

类：

```javascript
class User {
  name: string;

  constructor(name: string) {
    this.name = name;
  }

  label(): string {
    return this.name;
  }
}
```

## 4. 值、转换与 toString

主要运行时类型：`number`、`string`、`boolean`、`null`、`undefined`、`array`、`object`、`function`、`class`、`promise`、`error`。

`typeof` 差异：

```javascript
typeof null; // "null"
typeof [];   // "array"
```

显式转换：

```javascript
String(value);
Number(text);
Boolean(value);
JSON.stringify(value, null, 2);
JSON.parse(text);
```

原生 `toString()`：

```javascript
"hello".toString();        // "hello"
(15).toString(16);         // "f"
true.toString();           // "true"
false.toString();          // "false"
null.toString();           // "null"
undefined.toString();      // "undefined"
[1, "a", true].toString(); // "1,a,true"
({ a: 1 }).toString();     // "{a: 1}"
```

对象自有 `toString` 优先：

```javascript
let obj = { toString: function() { return "custom"; } };
obj.toString(); // "custom"
```

建议：生成日志或拼接文本时优先用模板字符串；需要与 API 兼容时再用 `.toString()`。

## 5. 常用内置对象

- `console.log/info/warn/error/debug/assert/time/timeEnd/trace/count/table`
- `Math.abs/min/max/pow/sqrt/floor/ceil/round/trunc/clamp/lerp`
- `JSON.stringify/parse`
- `Object.keys/values/entries/assign/hasOwn/freeze/seal/create`
- `Array.isArray/of/from`
- 数组方法：`push/pop/map/filter/reduce/find/includes/slice/splice/join/sort/reverse/flat`
- 字符串方法：`trim/includes/slice/split/replace/replaceAll/toUpperCase/toLowerCase/at`
- `Date`、`RegExp`、`Promise`、`Map`、`Set`、`Error/TypeError/RangeError/ReferenceError/SyntaxError`
- 计时器：`setTimeout/clearTimeout/setInterval/clearInterval/queueMicrotask/sleep`

## 6. 模块

简单脚本优先 `require`：

```javascript
let fs = require("@std/fs");
let local = require("./local"); // 可省略 .gs
```

库代码可用 ESM 风格：

```javascript
export const name = "tools";
export function label(x) { return name + ":" + x; }
export default 6;
```

```javascript
import fallback, { name, label as makeLabel } from "./tools.gs";
import * as tools from "./tools.gs";
```

解析要点：

- `@std/...`：原生模块。
- `./x`、`../x`、绝对路径：源文件或目录；会尝试 `.gs`、目录入口、`index.gs`。
- `name` / `name/subpath`：通过 `project.toml` 的 `[dependencies]` 与 `[exports]`。
- `@/*` 等别名：通过 `[imports]`。

## 7. 原生标准库选型

| 任务 | 模块 |
|------|------|
| 文件/目录 | `@std/fs` |
| 路径 | `@std/path` |
| 系统信息 | `@std/os` |
| 参数/env/cwd/exit | `@std/process` |
| 执行命令 | `@std/exec` |
| PTY/终端 | `@std/pty`, `@std/terminal` |
| HTTP 请求 | `@std/net/http/client` |
| HTTP/Web 服务 | `@std/net/http/server`, `@std/web` |
| TCP/WebSocket/IP | `@std/net/socket/*`, `@std/net/ws/*`, `@std/net/ip` |
| URL/MIME/Mail | `@std/url`, `@std/mime`, `@std/mail` |
| TOML/YAML/XML/CSV | `@std/toml`, `@std/yaml`, `@std/xml`, `@std/encoding/csv` |
| Base64/Hex/Buffer | `@std/encoding/base64`, `@std/encoding/hex`, `@std/buffer` |
| Hash/Crypto | `@std/hash`, `@std/crypto` |
| gzip/zip | `@std/compress/gzip`, `@std/archive/zip` |
| 模板/时间/日志 | `@std/template`, `@std/time`, `@std/log` |
| 事件/流/SSE/信号 | `@std/events`, `@std/stream`, `@std/sse`, `@std/signal` |
| 数据库 | `@std/db` |

文件示例：

```javascript
let fs = require("@std/fs");
let path = require("@std/path");
let os = require("@std/os");
let crypto = require("@std/crypto");

let root = path.join(os.tmpdir(), "gs-" + crypto.randomUUID());
fs.mkdirSync(root, { recursive: true });
try {
  let file = path.join(root, "out.txt");
  fs.writeTextSync(file, "hello");
  console.log(fs.readTextSync(file));
} finally {
  fs.rmSync(root, { recursive: true, force: true });
}
```

命令示例：

```javascript
let exec = require("@std/exec");

let r = exec.run("go", ["test", "./..."]);
if (!r.success) {
  console.error(r.stderr);
  throw new Error("exit " + String(r.exitCode));
}
console.log(r.stdout);
```

Web 示例：

```javascript
let web = require("@std/web");
let app = web.createApp();

app.get("/health", function(req, res) {
  res.json({ ok: true });
});

app.listen(8080);
console.log("listening on http://127.0.0.1:8080");
```

## 8. AI 安全规则

- 文件删除只限明确目录；临时目录用 `os.tmpdir()` + `crypto.randomUUID()`。
- 外部命令优先 `exec.run(cmd, argsArray)`，不要拼 shell 字符串。
- 网络请求必须处理非 2xx、超时和错误体。
- 服务脚本说明端口和退出方式。
- 长任务提醒使用更长 `--timeout` 或 `--timeout 0`。
- 输出给机器消费时用 JSON。

## 9. 生成前自检

- 没有 `==` / `!=`。
- 没有 `switch/case/default`。
- 没有 `eval/with/arguments`。
- 字符串拼接没有混合类型。
- 变量都已声明。
- `sort` 有比较函数。
- 文件、命令、网络副作用可控。
- 优先使用 `@std` 模块。
- 不确定 API 时查 `docs/builtins.md` 或 `examples/`。

