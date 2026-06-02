# GoScript builtins 文档逐项测试报告

> 基于 `docs/builtins.md` 对当前运行时与标准库进行逐项核对。
> 本报告区分“已自动测试覆盖”和“文档列出但当前未实现/规划项”，避免把目标 API 误当成已交付 API。

## 测试入口

- 解释器内置对象测试：`internal/evaluator/evaluator_test.go`
  - `TestEval_BuiltinsDocs_GlobalConsoleMathJSON`
  - `TestEval_BuiltinsDocs_ObjectAndArray`
  - `TestEval_BuiltinsDocs_PromiseStaticAll`
  - `TestEval_BuiltinsDocs_ExtendedGlobalMathObjectArrayStringNumber`
  - `TestEval_BuiltinsDocs_JSONStringMatchAllBooleanWrapper`
  - `TestEval_BuiltinsDocs_MapSet`
  - `TestEval_BuiltinsDocs_PromiseRaceAndAllSettled`
  - `TestEval_BuiltinsDocs_TimersAndMicrotasks`
  - Date / RegExp / console 相关 builtins 覆盖测试
  - 既有测试：`TestEval_StringMethods`、`TestEval_ErrorObjectFields`、`TestEval_ErrorObjectStack`、Promise 链式测试等
- 标准模块脚本级测试：`cmd/gs/main_test.go`
  - `@std/fs`、`@std/path`、`@std/process`、`@std/os`
  - `@std/toml`、`@std/yaml`、`@std/xml`
  - `@std/exec`、`@std/pty`、`@std/terminal`
  - `@std/stream`、`@std/sse`、`@std/net/http/client`
  - `@std/db` SQLite 基础与事务/预编译语句
  - `@std/crypto`、`@std/schema`
  - `@std/url`、`@std/buffer`、`@std/events`、`@std/timers`

## 逐项覆盖结果

| 章节 | 文档项 | 当前状态 | 自动测试 |
|------|--------|----------|----------|
| 1 | `print`、`println` | 已实现 | 已覆盖返回值与调用 |
| 1 | `require` | 已实现，CLI runner 注册 | 已覆盖相对模块、缓存、原生模块 |
| 1 | `setTimeout`、`setInterval` | 已实现基础调度；返回 `TimerId`，支持传参 | 已覆盖 |
| 1 | `clearTimeout`、`clearInterval` | 已实现基础取消 | 已覆盖 |
| 1 | `queueMicrotask` | 已实现基础微任务调度 | 已覆盖 |
| 1 | `parseInt`、`parseFloat`、`isNaN`、`isFinite` | 已实现；`parseInt` 支持 radix | 已覆盖 |
| 1 | `encodeURI`、`decodeURI`、`encodeURIComponent`、`decodeURIComponent` | 已实现基础百分号编解码 | 已覆盖 |
| 2 | `console.log/info/warn/error/debug/assert/time/timeEnd/trace/count/countReset/group/groupEnd/table` | 已实现基础输出/状态行为 | 已覆盖 |
| 3 | Math 常量与文档列出的函数 | 已实现：`E/LN2/LN10/LOG2E/LOG10E/PI/SQRT2/SQRT1_2` 与 `abs/sign/floor/ceil/round/trunc/min/max/pow/sqrt/cbrt/exp/log/log2/log10/sin/cos/tan/asin/acos/atan/atan2/random/hypot/clamp/lerp` | 已覆盖 |
| 4 | `JSON.stringify(value, replacer?, space?)`、`JSON.parse(text, reviver?)` | 已实现基础值、数组、对象，支持函数型 `replacer/reviver` 与缩进 `space` | 已覆盖 |
| 5 | Object 文档列出的基础方法 | 已实现：`create/assign/keys/values/entries/fromEntries/freeze/isFrozen/seal/isSealed/getPrototypeOf/setPrototypeOf/hasOwn/is/defineProperty/getOwnPropertyDescriptor/getOwnPropertyNames`；描述符与冻结/封闭为浅层基础版 | 已覆盖 |
| 6 | Array 实例属性与方法 | 已实现文档列出的实例项；`sort` 当前要求显式比较函数 | 已覆盖 |
| 6 | `Array.isArray/of/from` | 已实现基础版；`Array.from` 支持数组、字符串、数组式对象和 mapFn | 已覆盖 |
| 7 | String 实例与静态方法 | 已实现：静态 `fromCharCode/fromCodePoint`，实例 `length/charAt/charCodeAt/codePointAt/concat/includes/indexOf/lastIndexOf/startsWith/endsWith/slice/substring/split/replace/replaceAll/trim/trimStart/trimEnd/toUpperCase/toLowerCase/padStart/padEnd/repeat/normalize/match/matchAll/search/at/isWellFormed/toWellFormed`；`normalize` 当前返回原字符串 | 已覆盖 |
| 7 | `String.raw` | 已实现基础调用形态：读取 `{ raw: [...] }` 并交织替换值 | 已覆盖 |
| 8 | Number 静态属性、静态方法、实例方法 | 已实现文档列出项：安全整数边界、浮点常量、Infinity/NaN、`isInteger/isFinite/isNaN/isSafeInteger/parseFloat/parseInt`、`toString/toFixed/toPrecision/toExponential` | 已覆盖 |
| 9 | Boolean wrapper | 已实现基础版：`new Boolean(...).valueOf()`、`toString()`；全局 `Boolean()` 保持原始布尔转换 | 已覆盖 |
| 10 | Date | 已实现基础版：构造、`Date.now/parse/UTC`、本地与 UTC getter/setter、`getTime/valueOf/toISOString/toLocale*` | 已覆盖 |
| 11 | RegExp | 已实现基础版：构造、`test/exec`、`source/flags/global/ignoreCase`；字符串 `match/search/replace` 已支持 RegExp 基础协作 | 已覆盖 |
| 12 | `new Promise`、`resolve`、`reject`、`all`、实例 `then/catch/finally` | 已实现 | 已覆盖 |
| 12 | `Promise.race`、`Promise.allSettled` | 已实现基础版 | 已覆盖 |
| 13 | Map / Set | 已实现基础版：构造、初始化、`set/get/has/delete/clear/add/size` | 已覆盖 |
| 14 | `Error`、`TypeError`、`RangeError`、`ReferenceError`、`SyntaxError` 与 `name/message/stack` | 已实现 | 已覆盖 |
| 15 | `@std/fs` | 已实现文档列出的同步 API | 已覆盖 |
| 15 | `@std/path` | 已实现文档列出的 API | 已覆盖 |
| 15 | `@std/os` | 已实现文档列出的 API | 已覆盖 |
| 15 | `@std/process` | 已实现文档列出的 API | 已覆盖 |
| 15 | `@std/toml`、`@std/yaml`、`@std/xml` | 已实现 | 已覆盖 |
| 15 | `@std/exec` | 已实现 `run/output/combinedOutput/start/command/spawn` 与交互流 | 已覆盖 |
| 15 | `@std/pty`、`@std/terminal` | 已实现 | 已覆盖 |
| 15 | `@std/stream`、`@std/sse`、`@std/net/http/client.stream` | 已实现 L3 streaming/SSE/stream 抽象 | 已覆盖 |
| 15 | `@std/db` | 已实现 SQLite、PostgreSQL、MySQL、MSSQL 驱动入口；SQLite 使用 `modernc.org/sqlite` nocgo | SQLite 已覆盖 |
| 15 | `@std/url`、`@std/buffer`、`@std/events`、`@std/timers` | 已实现基础版：URL/SearchParams、Buffer 编码、EventEmitter、timer 模块别名 | 已覆盖 |
| 15 | 文档中的裸 `net/http/crypto/events/timers/buffer` Node 风格模块 | 当前原生模块统一使用 `@std/*` 命名；裸模块名待文档修订或 resolver 别名 | 待文档修订 |
| 16 | Go 嵌入示例 | 文档示例与当前 API 形态不完全一致 | 待文档修订 |

## 本轮新增自动测试覆盖

- 全局转换和工具函数：`String`、`Number`、`Boolean`、`parseInt`、`parseFloat`、`isNaN`、`isFinite`。
- 控制台基础：`console.log` 返回 `undefined`。
- Math 基础：`PI`、`abs`、`floor`、`ceil`、`round`、`max`、`min`、`pow`、`sqrt`、`random`。
- JSON 基础：数组/对象序列化和对象/数组解析。
- Object 基础：`keys`、`values`、`entries`、`assign`、`hasOwn`。
- Array 文档中的所有当前实例方法：增删、切片、查找、高阶函数、排序、拍平、填充、内部复制等。
- String 当前实例方法补充：`charCodeAt`、负索引 `slice`、带 limit 的 `split`。
- Promise 静态 `all` 聚合。

## 本轮继续补齐

- 全局 URI：`encodeURI/decodeURI/encodeURIComponent/decodeURIComponent`。
- Math：补齐文档列出的常量和函数。
- Object：补齐 `create/fromEntries/freeze/seal/prototype/is/defineProperty/descriptor/names` 等基础能力。
- Array：补齐 `Array.isArray/of/from`。
- String：补齐静态 `fromCharCode/fromCodePoint` 与一批实例方法。
- Number：补齐静态常量、静态判断/解析方法和实例格式化方法。

## 多 worker 并行补齐

- Worker A：补齐 `Promise.race/allSettled`、`clearTimeout/clearInterval`、`queueMicrotask` 和 `TimerId`。
- Worker B：补齐完整 console 章节的基础行为。
- Worker C：补齐 Map / Set 基础版。
- Worker D：补齐 Date / RegExp 基础版，并让 String 与 RegExp 基础协作。

## 本轮增强

- JSON：补齐函数型 `replacer/reviver` 与 `space` 缩进输出。
- String：补齐 `matchAll`，并增强 `replaceAll(RegExp, replacement)`。
- Boolean：补齐 `new Boolean(...).valueOf()` 与 `toString()` 基础 wrapper。

## 后续补齐建议

1. 先拆分 `docs/builtins.md`：把“当前已实现 API”和“路线图/目标 API”分开，避免使用者误判。
2. 第一优先级补强 RegExp/Map/Set 的完整语义：RegExp flags 全量兼容、迭代协议细节。
3. 第二优先级修正文档中的标准模块命名：当前网络模块使用 `@std/net/http/client`、`@std/net/http/server`、`@std/net/socket/*`、`@std/net/ws/*`，URL/Buffer/Events/Timers 使用 `@std/url`、`@std/buffer`、`@std/events`、`@std/timers`。
4. Go 嵌入章节需要按当前 `lexer/parser/evaluator/object/module` API 重写可运行示例。
