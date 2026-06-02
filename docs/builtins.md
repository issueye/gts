# GoScript 内置对象与标准库

> 本文档列出 v0.1 提供的全局对象、内置函数和标准库模块。
> 所有内置对象在脚本启动时已挂载到全局环境，可直接使用。

---

## 1. 全局函数

| 函数 | 签名 | 行为 |
|------|------|------|
| `print(...values)` | `...any → undefined` | 输出到 stdout，**不带**换行 |
| `println(...values)` | `...any → undefined` | 输出到 stdout，**带** `\n`（`console.log` 的别名） |
| `require(path)` | `string → any` | 加载并执行模块，返回 `module.exports`（v0.1 简化） |
| `setTimeout(fn, ms, ...args)` | `(fn, number, ...any) → TimerId` | 延迟执行 |
| `clearTimeout(id)` | `(TimerId) → undefined` | 取消 |
| `setInterval(fn, ms, ...args)` | `(fn, number, ...any) → TimerId` | 周期执行 |
| `clearInterval(id)` | `(TimerId) → undefined` | 取消 |
| `queueMicrotask(fn)` | `(fn) → undefined` | 注册一个微任务 |
| `parseInt(s, radix?)` | `(string, number?) → number` | 解析整数 |
| `parseFloat(s)` | `(string) → number` | 解析浮点数 |
| `isNaN(v)` | `(any) → boolean` | 严格的 NaN 判断 |
| `isFinite(v)` | `(any) → boolean` | 是否有限数 |
| `encodeURI(s)` / `decodeURI(s)` | `(string) → string` | URL 编/解码 |
| `encodeURIComponent` / `decodeURIComponent` | `(string) → string` | 组件级编/解码 |

---

## 2. `console`

```javascript
console.log(...values);     // 普通
console.info(...values);    // 带 [INFO] 前缀
console.warn(...values);    // 带 [WARN] 前缀，输出到 stderr
console.error(...values);   // 带 [ERROR] 前缀，输出到 stderr
console.debug(...values);   // 默认隐藏
console.assert(cond, ...);  // 断言失败时打印
console.time(label);        // 开始计时
console.timeEnd(label);     // 结束计时并打印
console.trace(...values);   // 打印当前堆栈
console.count(label?);      // 计数
console.countReset(label?); // 重置计数
console.group(label?);      // 缩进
console.groupEnd();         // 取消缩进
console.table(arrOrObj);    // 表格化打印
```

---

## 3. `Math`

### 3.1 常量

| 属性 | 值 |
|------|------|
| `Math.E` | `2.718281828459045` |
| `Math.LN2` | `0.6931471805599453` |
| `Math.LN10` | `2.302585092994046` |
| `Math.LOG2E` | `1.4426950408889634` |
| `Math.LOG10E` | `0.4342944819032518` |
| `Math.PI` | `3.141592653589793` |
| `Math.SQRT2` | `1.4142135623730951` |
| `Math.SQRT1_2` | `0.7071067811865476` |

### 3.2 函数

| 方法 | 说明 |
|------|------|
| `abs(x)` | 绝对值 |
| `sign(x)` | 符号 -1 / 0 / 1 |
| `floor(x)` `ceil(x)` `round(x)` `trunc(x)` | 向下/上/四舍五入/截断 |
| `min(...xs)` `max(...xs)` | 多参最小/最大 |
| `pow(b, e)` | 幂 |
| `sqrt(x)` `cbrt(x)` | 立方根 |
| `exp(x)` | `e^x` |
| `log(x)` `log2(x)` `log10(x)` | 自然/2/10 对数 |
| `sin(x)` `cos(x)` `tan(x)` | 三角（弧度） |
| `asin(x)` `acos(x)` `atan(x)` `atan2(y, x)` | 反三角 |
| `random()` | `[0, 1)` 伪随机 |
| `hypot(...xs)` | 欧氏范数 |
| `clamp(x, lo, hi)` | 范围限制（扩展） |
| `lerp(a, b, t)` | 线性插值（扩展） |

---

## 4. `JSON`

| 方法 | 行为 |
|------|------|
| `JSON.stringify(value, replacer?, space?)` | 序列化为字符串。`replacer` 接受 key/value 返回替换值；`space` 是缩进数或字符串 |
| `JSON.parse(text, reviver?)` | 解析为对象。`reviver` 在每个 key/value 上调用 |

```javascript
JSON.stringify({a: 1, b: [1, 2]});
// '{"a":1,"b":[1,2]}'

JSON.stringify({a: 1}, null, 2);
// '{\n  "a": 1\n}'

JSON.parse('{"a":1}').a;  // 1
```

支持的 JSON 值：`null`、`true`/`false`、数字、字符串、数组、对象。

---

## 5. `Object`

| 方法 | 签名 | 行为 |
|------|------|------|
| `Object.create(proto, props?)` | `(object?, object?) → object` | 以 `proto` 为原型创建 |
| `Object.assign(target, ...src)` | `(...object) → object` | 浅合并 |
| `Object.keys(o)` | `(object) → string[]` | 自身可枚举字符串键 |
| `Object.values(o)` | `(object) → any[]` | 自身可枚举值 |
| `Object.entries(o)` | `(object) → [string, any][]` | 自身键值对 |
| `Object.fromEntries(iter)` | `(Iterable) → object` | 反向构造 |
| `Object.freeze(o)` | `(object) → object` | 浅冻结，属性不可改不可删 |
| `Object.isFrozen(o)` | `(object) → boolean` | 是否冻结 |
| `Object.seal(o)` | `(object) → object` | 浅封闭 |
| `Object.isSealed(o)` | `(object) → boolean` | 是否封闭 |
| `Object.getPrototypeOf(o)` | `(object) → object?` | 取原型 |
| `Object.setPrototypeOf(o, p)` | `(object, object?) → object` | 设原型 |
| `Object.hasOwn(o, k)` | `(object, string) → boolean` | 自身属性判断 |
| `Object.is(a, b)` | `(any, any) → boolean` | 严格同值（`NaN` 视为相等） |
| `Object.defineProperty(o, k, desc)` | `...→ object` | 精细定义 |
| `Object.getOwnPropertyDescriptor(o, k)` | `...→ object?` | 取属性描述符 |
| `Object.getOwnPropertyNames(o)` | `...→ string[]` | 自身所有字符串键 |

---

## 6. `Array`

### 6.1 静态方法

| 方法 | 行为 |
|------|------|
| `Array.isArray(v)` | 是否为数组 |
| `Array.of(...xs)` | 由参数构造数组 |
| `Array.from(iterable, mapFn?)` | 由可迭代对象构造 |

### 6.2 实例方法

| 方法 | 签名 | 说明 |
|------|------|------|
| `length` | 数字 | 当前长度 |
| `push(...x)` | `...T → number` | 末尾追加，返回新长度 |
| `pop()` | `() → T?` | 末尾弹出 |
| `shift()` `unshift(...x)` | `() → T?` / `...T → number` | 头部 |
| `concat(...arrs)` | `...Array<T> → Array<T>` | 拼接（不修改自身） |
| `slice(start?, end?)` | `(number?, number?) → Array<T>` | 切片 |
| `splice(start, deleteCount?, ...items)` | 返回被删元素 | 修改自身 |
| `indexOf(v, from?)` | `(T, number?) → number` | 首次索引（无 → -1） |
| `lastIndexOf(v)` | 同上反向 | |
| `includes(v)` | `(T) → boolean` | 是否包含 |
| `find(fn)` | `((T) → boolean) → T?` | 首匹配 |
| `findIndex(fn)` | `((T) → boolean) → number` | 首匹配索引 |
| `filter(fn)` | `((T) → boolean) → Array<T>` | 过滤 |
| `map(fn)` | `((T) → U) → Array<U>` | 映射 |
| `reduce(fn, init?)` | `((acc, T) → acc, acc?) → acc` | 归约 |
| `reduceRight` | 同 reduce，从右 | |
| `forEach(fn)` | `((T, number) → void) → undefined` | 遍历 |
| `some(fn)` `every(fn)` | 谓词短路 | |
| `sort(cmp?)` | `((a, b) → number) → Array<T>` | 排序（修改自身） |
| `reverse()` | 反转（修改自身） | |
| `join(sep?)` | `(string?) → string` | 默认 `","` |
| `flat(depth?)` | `Array<Array<T>> → Array<T>` | 拍平 |
| `flatMap(fn)` | 映射后拍平一层 | |
| `fill(v, start?, end?)` | 填充 | |
| `copyWithin(target, start, end?)` | 内部复制 | |

---

## 7. `String`

### 7.1 静态方法

| 方法 | 行为 |
|------|------|
| `String.fromCharCode(...codes)` | 由 UTF-16 码点构造 |
| `String.fromCodePoint(...points)` | 由 Unicode 码点构造 |
| `String.raw\`...${}...\`` | 模板原始形式（不解析转义） |

### 7.2 实例属性与方法

| 方法 | 行为 |
|------|------|
| `length` | 字符数（Unicode 码点数） |
| `charAt(i)` `charCodeAt(i)` `codePointAt(i)` | 取字符/码点 |
| `concat(...s)` | 拼接 |
| `indexOf(s, from?)` `lastIndexOf(s)` | 子串位置 |
| `includes(s)` `startsWith(s)` `endsWith(s)` | 包含判断 |
| `slice(start?, end?)` `substring(start?, end?)` | 切片 |
| `split(sep, limit?)` | 分割为数组 |
| `replace(pattern, repl)` | 首次替换；`repl` 可为函数 |
| `replaceAll(pattern, repl)` | 全量替换 |
| `trim()` `trimStart()` `trimEnd()` | 去除空白 |
| `toUpperCase()` `toLowerCase()` | 大小写 |
| `padStart(len, pad?)` `padEnd(len, pad?)` | 填充 |
| `repeat(n)` | 重复 |
| `normalize(form?)` | Unicode 正规化 |
| `match(re)` `matchAll(re)` `search(re)` | 正则匹配 |
| `at(i)` | 相对索引（支持负数） |
| `isWellFormed()` `toWellFormed()` | 单独代理项处理 |

---

## 8. `Number`

### 8.1 静态属性

| 属性 | 值 |
|------|------|
| `Number.MAX_SAFE_INTEGER` | `2^53 - 1` |
| `Number.MIN_SAFE_INTEGER` | `-(2^53 - 1)` |
| `Number.MAX_VALUE` | 最大正浮点 |
| `Number.MIN_VALUE` | 最小正浮点 |
| `Number.EPSILON` | `2.220446049250313e-16` |
| `Number.POSITIVE_INFINITY` | `Infinity` |
| `Number.NEGATIVE_INFINITY` | `-Infinity` |
| `Number.NaN` | `NaN` |

### 8.2 静态方法

| 方法 | 行为 |
|------|------|
| `Number.isInteger(v)` | 是否整数 |
| `Number.isFinite(v)` | 是否有限 |
| `Number.isNaN(v)` | 是否 NaN（不转换类型） |
| `Number.isSafeInteger(v)` | 是否安全整数 |
| `Number.parseFloat(s)` | 等价全局 `parseFloat` |
| `Number.parseInt(s, r?)` | 等价全局 `parseInt` |
| `Number.isFinite` | 同上 |

### 8.3 实例方法

| 方法 | 行为 |
|------|------|
| `toString(radix?)` | 转字符串，可指定进制 |
| `toFixed(d)` | 定点小数 |
| `toPrecision(p)` | 有效位 |
| `toExponential(d)` | 科学计数 |

---

## 9. `Boolean`

```javascript
new Boolean(true).valueOf();   // true
new Boolean(false).toString(); // "false"
```

> 与 JS 一致，**避免**用 `new Boolean` 作为真值判断。

---

## 10. `Date`

```javascript
let d = new Date();                  // 当前时间
let d2 = new Date(2024, 0, 1);       // 年月日（月份 0 开始）
let d3 = new Date("2024-01-01");     // ISO 字符串
d.getFullYear();
d.getMonth();     // 0-11
d.getDate();
d.getHours(); d.getMinutes(); d.getSeconds();
d.getTime();      // 毫秒时间戳
d.getDay();       // 0-6, 周日=0
d.toISOString();
d.toLocaleString();
d.toLocaleDateString();
d.toLocaleTimeString();
d.setFullYear(y); d.setMonth(m); d.setDate(d);
d.setHours(h); d.setMinutes(m); d.setSeconds(s);
d.setTime(t);
d.valueOf();
```

---

## 11. `RegExp`

```javascript
let re = /ab+c/gi;
re.test("abbbbc");   // true
re.exec("abbbbc");   // ["abbbbc"]
"abc".match(/a(b)/); // ["ab", "b"]
"a1b2".replace(/\d/g, "*"); // "a*b*"
```

支持的特性：
- 标志：`g` `i` `m` `s` `u` `y`
- 元字符：`. \w \W \s \S \d \D \b \B ^ $`
- 量词：`? * + {n} {n,} {n,m}`，附 `?` / `+` 修饰
- 字符类：`[abc] [a-z] [^abc]`
- 分组：`(...)` `(?:...)` `(?<name>...)`
- 反向引用：`\1 \2 ... \k<name>`
- 断言：`^ $ \b \B (?=...) (?!...) (?<=...) (?<!...)`

实现策略：调用 Go 的 `regexp/syntax` + `regexp` 包，对其接口做薄包装。

---

## 12. `Promise`

```javascript
new Promise(executor);                  // executor: (resolve, reject) => void
Promise.resolve(value);                 // 已兑现
Promise.reject(reason);                 // 已拒绝
Promise.all(iterable);                  // 全成功才成功，失败取首个原因
Promise.race(iterable);                 // 首个决断
Promise.allSettled(iterable);           // 全部决断
```

```go
// 内部
type Promise struct { ... }
```

---

## 13. `Map` / `Set`（v0.1 基础版）

```javascript
let m = new Map();
m.set("k", 1);
m.get("k");          // 1
m.has("k");          // true
m.delete("k");
m.size;
m.clear();
for (let [k, v] of m) { /* ... */ }

let s = new Set([1, 2, 3]);
s.add(4);
s.has(2);            // true
s.delete(2);
s.size;
for (let v of s) { /* ... */ }
```

---

## 14. `Error`

```javascript
new Error("msg");
new TypeError("type wrong");
new RangeError("out of range");
new ReferenceError("undefined var");
new SyntaxError("bad syntax");

err.message;
err.name;
err.stack;
```

---

## 15. 标准模块（v0.1 起步）

通过 `require("/abs/path")` 或 `import x from "./relative.gs"` 加载。

| 模块 | 提供 |
|------|------|
| `@std/fs` | `readFileSync`, `readTextSync`, `writeFileSync`, `writeTextSync`, `appendFileSync`, `writeFileAtomicSync`, `existsSync`, `readdirSync`, `walkSync`, `globSync`, `copyFileSync`, `rmSync`, `mkdtempSync`, `realpathSync`, `lstatSync`, `mkdirSync`, `statSync`, `renameSync`, `unlinkSync` |
| `@std/path` | `join`, `resolve`, `relative`, `normalize`, `dirname`, `basename`, `extname`, `isAbs`, `toSlash`, `fromSlash`, `matches`, `parse`, `format`, `splitList`, `sep`, `delimiter` |
| `@std/os` | `platform`, `arch`, `eol`, `type`, `release`, `hostname`, `cpus`, `homedir`, `tmpdir`, `userInfo` |
| `net` | `fetch`（基于 Go `net/http`），`Server` 类（TCP） |
| `http` | `createServer`, `request`, `get` |
| `url` | `URL` 类，`URLSearchParams` 类 |
| `crypto` | `randomBytes`, `createHash`（md5/sha1/sha256/sha512），`createHmac` |
| `events` | `EventEmitter` 类 |
| `timers` | `setTimeout` 等的模块化别名 |
| `buffer` | `Buffer` 类（与 Node 类似的字节缓冲区） |
| `@std/process` | `argv`, `argv0`, `env`, `envObject`, `pid`, `cwd`, `chdir`, `execPath`, `getenv`, `setenv`, `unsetenv`, `uptime`, `hrtime`, `version`, `exit` |

> 模块 API 设计参考 Node.js CommonJS 子集 + 必要扩展。

---

## 16. Go 嵌入

```go
// 在 Go 端注册自定义对象
e := evaluator.New()
e.RegisterGlobal("MyGoStruct", &object.Instance{
    Class: &object.Class{Name: "MyGoStruct"},
    Props: map[string]object.Object{
        "sayHello": &object.Builtin{
            Fn: func(env *evaluator.Env, args ...object.Object) object.Object {
                name := args[0].(*object.String).Value
                return &object.String{Value: "Hello, " + name}
            },
        },
    },
})
```

详情见 [`docs/design.md`](design.md) 第 12 节。
