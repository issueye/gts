# GoScript AI 使用指南

> 给 AI Agent 生成 `.gs` 脚本时使用的紧凑上下文。  
> 详细规范见 [`language-spec.md`](language-spec.md)、[`grammar.ebnf`](grammar.ebnf)、[`builtins.md`](builtins.md)、[`async-model.md`](async-model.md)。

## 1. 一句话定位

GoScript 是 JS 风格的动态脚本语言，但比 JS 更严格，适合写文件处理、系统自动化、HTTP 服务、数据转换和本地工具脚本。

生成代码时优先遵守这几条：

- 文件后缀是 `.gs`。
- 变量必须先声明：用 `let` 或 `const`。
- 只用 `===` / `!==`，不要生成 `==` / `!=`。
- 不使用 `switch/case/default`，改用 `match`。
- 不使用 `eval`、`with`、`arguments`。
- `+` 不做混合类型拼接；数字转字符串用模板字符串或 `String(value)`。
- 数组排序必须传比较函数：`arr.sort((a, b) => a - b)`。
- 文件、命令、网络、数据库等能力优先使用 `@std/...` 原生库。

最小脚本：

```javascript
function main() {
  console.log("hello");
}

main();
```

## 2. 运行与查 API

常用运行命令：

```bash
.\gs.exe main.gs
.\gs.exe --timeout 30s main.gs
.\gs.exe --timeout 0 main.gs
.\gs.exe --workers 1 main.gs
.\gs.exe --check-types main.gs  # 开启可用的类型检查能力
```

项目入口：

```toml
[project]
name = "demo"
version = "0.1.0"
entry = "main.gs"
```

```bash
.\gs.exe run
```

查看原生库接口、参数和中文说明：

```bash
.\gs.exe --api_doc all
.\gs.exe --api_doc "@std/web"
.\gs.exe --api_doc "@std/fs"
```

AI 生成涉及原生库的代码前，必须按这个顺序查询：

1. 不确定模块名时先运行 `.\gs.exe --api_doc all`，从输出里选择最匹配的 `@std/...` 模块。
2. 对选中的每个模块运行 `.\gs.exe --api_doc "<module>"`。
3. 只使用文档输出中存在的方法名、参数形式和返回值；不要凭 JS/Node/Deno 经验猜接口。
4. 同一任务涉及多个原生库时，分别查询所有相关模块，例如文件 + 路径要同时查 `@std/fs` 和 `@std/path`。
5. 如果参数不确定，优先改用 `--api_doc` 中更明确的调用形式，或生成前再次查询目标模块。

其它命令：

```bash
.\gs.exe init [dir]
.\gs.exe pack [dir] [out.gspkg]
.\gs.exe dist [dir] [out]
.\gs.exe bundle <entry.gs> [out.gs]
```

## 3. 语法速查

变量、函数和控制流：

```javascript
let count = 1;
const name = "Ada";

function add(a: number, b: number): number {
  return a + b;
}

let twice = x => x * 2;

if (count > 0) {
  println("positive");
} else {
  println("other");
}

for (let i = 0; i < 3; i++) {}
for (let k in obj) {} // key / index
for (let v of arr) {} // value
while (count < 10) {}
```

模式匹配：

```javascript
let label = match status {
  200 (val) => "OK",
  404 (val) => "Not Found",
  500..599 (val) => "Server Error",
  _ => "Unknown",
};
```

错误处理：

```javascript
try {
  risky();
} catch (e) {
  console.error(e.name + ": " + e.message);
} finally {
  cleanup();
}
```

类型注解是可选的：

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

## 4. 值、转换与字符串化

主要运行时类型：

```text
number, string, boolean, null, undefined, array, object,
function, class, promise, error
```

`typeof` 和 JS 有差异：

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

建议：日志、错误信息、路径拼接优先用模板字符串；与外部 API 兼容时再使用 `.toString()`。

## 5. 模块写法

简单脚本优先用 `require`：

```javascript
let local = require("./local"); // 可省略 .gs
```

库代码可以使用 ESM 风格：

```javascript
export const name = "tools";
export function label(x) { return name + ":" + x; }
export default 6;
```

```javascript
import fallback, { name, label as makeLabel } from "./tools.gs";
import * as tools from "./tools.gs";
```

解析规则：

- `@std/...`：原生标准库模块。
- `./x`、`../x`、绝对路径：源文件或目录；会尝试 `.gs`、目录入口、`index.gs`。
- `name` / `name/subpath`：通过 `project.toml` 的 `[dependencies]` 与 `[exports]`。
- `@/*` 等别名：通过 `[imports]`。

## 6. 原生库使用规则

本指南不维护原生库方法清单。原生库接口以 `--api_doc` 输出为准，包含模块名、方法名、参数、返回值和中文说明。

AI 生成代码时遵守：

- 需要文件、命令、网络、数据库、路径、系统信息等能力时，优先使用 `@std/...` 原生库。
- 写 `require("@std/...")` 或 `import` 前，先用 `--api_doc all` 确认模块路径。
- 调用任何原生库方法前，先用 `--api_doc "<module>"` 确认签名。
- 不要把 Node.js、浏览器或其它语言的标准库接口当作 GoScript 原生库接口。
- 不要编造未出现在 `--api_doc` 输出里的 options 字段、返回字段或方法别名。
- 生成示例代码时，只展示已经由 `--api_doc` 确认过的调用。

常用查询模式：

```bash
# 先看支持哪些原生库模块
.\gs.exe --api_doc all

# 再查目标模块的完整接口
.\gs.exe --api_doc "@std/web"
.\gs.exe --api_doc "@std/net/http/client"
.\gs.exe --api_doc "@std/fs"
```

如果用户只描述任务、没有给出模块名，AI 应先根据 `--api_doc all` 的模块列表选择候选模块，再查询候选模块接口，最后生成代码。

## 7. 内置对象速查

- `console.log/info/warn/error/debug/assert/time/timeEnd/trace/count/table`
- `Math.abs/min/max/pow/sqrt/floor/ceil/round/trunc/clamp/lerp`
- `JSON.stringify/parse`
- `Object.keys/values/entries/assign/hasOwn/freeze/seal/create`
- `Array.isArray/of/from`
- 数组方法：`push/pop/map/filter/reduce/find/includes/slice/splice/join/sort/reverse/flat`
- 字符串方法：`trim/includes/slice/split/replace/replaceAll/toUpperCase/toLowerCase/at`
- `Date`、`RegExp`、`Promise`、`Map`、`Set`
- 错误类型：`Error`、`TypeError`、`RangeError`、`ReferenceError`、`SyntaxError`
- 计时器：`setTimeout/clearTimeout/setInterval/clearInterval/queueMicrotask/sleep`

## 8. AI 生成安全规则

- 删除文件前必须限定目录；临时目录用 `os.tmpdir()` + `crypto.randomUUID()`。
- 外部命令优先使用 `@std/exec` 中已查询到的数组参数形式，不要拼接 shell 字符串。
- 网络请求必须处理非 2xx、超时和错误体。
- 服务脚本必须说明端口和退出方式。
- 长任务使用更长 `--timeout` 或 `--timeout 0`。
- 输出给机器消费时用 JSON，不要混入解释性文本。
- 数据库脚本优先使用参数数组，不要拼接用户输入到 SQL。

## 9. 生成前自检

生成 `.gs` 前逐项检查：

- 没有 `==` / `!=`。
- 没有 `switch/case/default`。
- 没有 `eval/with/arguments`。
- 没有混合类型字符串拼接。
- 变量都已声明。
- `sort` 有比较函数。
- 文件、命令、网络、数据库副作用可控。
- 使用原生库前已查询 `--api_doc`。
- 没有使用未出现在 `--api_doc` 输出里的原生库方法、参数或返回字段。
