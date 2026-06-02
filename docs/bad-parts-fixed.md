# GoScript vs JavaScript：被修复的"Bad Parts"

> 本文档系统地列出 JavaScript 中**广泛被认为反直觉、容易出错、或设计缺陷**的特性，
> 并说明 GoScript 的处理方式。
>
> 哲学：**保留 JS 的语法熟悉度，但消除那些连老手也会踩的坑。**

---

## 1. 类型比较与相等性

### 1.1 完全移除 `==` 与 `!=`（松散相等）

**JS 的问题**：

| 表达式 | 结果 | 合理吗？ |
|--------|------|---------|
| `0 == "0"` | `true` | ❌ |
| `0 == false` | `true` | ❌ |
| `"" == false` | `true` | ❌ |
| `[] == false` | `true` | ❌ |
| `[] == 0` | `true` | ❌ |
| `[0] == 0` | `true` | ❌ |
| `[1,2] == "1,2"` | `true` | ❌ |
| `null == undefined` | `true` | ⚠️ 见 1.3 |

JS 的 `==` 内部有一条 16 步的"抽象相等比较算法"，需要常年记忆。
几乎所有现代 Lint 工具（ESLint、TS）都禁止 `==` 和 `!=`。

**GoScript 决策**：
- `==` 与 `!=` 是**编译期错误**（语法阶段拒绝）。
- 唯一允许的相等运算符是 `===` 与 `!==`，**严格相等，无隐式转换**。
- 工具链不需要为这种经典坑提供 Lint 规则，因为从语言层面就不可能写出这种代码。

```javascript
// GoScript
1 === "1";   // false（类型不同）
1 !== "1";   // true
1 == "1";    // SyntaxError
1 != "1";    // SyntaxError
```

---

### 1.2 `NaN === NaN` 为 `true`

**JS 的问题**：IEEE-754 规定 NaN 不等于自身，JS 忠实实现：`NaN === NaN` 为 `false`。
这违反直觉（"一个东西怎么可能不等于自己？"），
使得 `===` 不能直接用来判断"两个数是否数学相等"，
开发者被迫用 `Object.is` 或 `Number.isNaN`。

**GoScript 决策**：
- `NaN === NaN` → `true`（与 IEEE-754 偏离，但符合直觉）
- `Object.is(NaN, NaN)` → `true`
- `Number.isNaN(NaN)` → `true`（仍保留这个工具函数用于与外部约定对齐）

---

### 1.3 `null` 与 `undefined` 是两个独立值

**JS 的问题**：
- `null == undefined` 为 `true`（`==` 专有的特殊规则）
- `null === undefined` 为 `false`
- `typeof null === "object"`（历史 bug）
- `Number(null) === 0`（即 `null` 在数字上下文中变 0）
- `Number(undefined) === NaN`

**GoScript 决策**：
- `null === undefined` → `false`（两个完全不同的字面量）
- `typeof null` → `"null"`（不是 `"object"`）
- `Number(null)` → `0`（保留）
- `Number(undefined)` → `NaN`（保留）
- `typeof undefined` → `"undefined"`

> 因为 `==` 已被移除，开发者也无需再记住 `null == undefined` 的怪规则。

---

### 1.4 `typeof` 对每种类型返回独立字符串

**JS 的问题**：`typeof` 不能区分 `null` 和 `object`，
不能区分 `array` 和 `object`，使得运行时类型判断非常啰嗦：
```javascript
Array.isArray(x) || (typeof x === "object" && x !== null)
```

**GoScript 决策**：

| 值 | `typeof` |
|----|----------|
| `42` | `"number"` |
| `"a"` | `"string"` |
| `true` | `"boolean"` |
| `null` | `"null"` |
| `undefined` | `"undefined"` |
| `{}` | `"object"` |
| `[]` | `"array"` |
| `function(){}` | `"function"` |
| `class C{}` | `"class"` |
| `new C()` | `"object"`（实例） |
| `new Promise(...)` | `"promise"` |
| `Symbol(...)` | `"symbol"`（v0.2） |

> 这样 `Array.isArray(x)` 仍然正确，但 `typeof x === "array"` 也能直接用于类型分流。

---

### 1.5 `+` 严格（混合类型 TypeError）

**JS 的问题**：
```javascript
"3" + 1;     // "31"   ← 字符串拼接赢
"3" - 1;     // 2      ← 反过来又走数值
"3" * "2";   // 6      ← 又走数值
1 + 2 + "3"; // "33"   ← 左结合导致不同结果
```

`+` 的行为不对称是 JS 公认的"最尖锐的设计错误"之一，
Linus Torvalds 在 2007 年就公开批评过。

**GoScript 决策**：
- `number + number` → `number`（加法）
- `string + string` → `string`（连接）
- 其他任何类型组合 → **`TypeError`**
- 显式拼接使用模板字符串 `` `${a}` `` 或 `String(a)`
- 显式数值化使用 `Number(a)`、`parseFloat(a)`

```javascript
// GoScript
1 + 2;          // 3
"a" + "b";      // "ab"
1 + "1";        // TypeError
"x" + null;     // TypeError
`${1}1`;        // "11"      ← 正确做法
String(1) + "1"; // "11"      ← 正确做法
```

> `-`、`*`、`/`、`%`、`**` 仍保留 JS 习惯（`Number(x)` 隐式转换），
> 因为它们没有"连接"这个备选语义，行为更可预测。

---

### 1.6 比较运算符要求类型一致

**JS 的问题**：
```javascript
"10" < "9";     // true  ← 字典序！
"10" < 9;       // false ← 数值比较，10 > 9
"   " < 9;      // true  ← 空白字符串转 0
"" < 0;         // false ← 0 < 0 是 false
null < 0;       // false
null >= 0;      // true   ← null 转 0
```

字符串与数字混用比较，行为高度依赖类型转换路径，调试时极痛苦。

**GoScript 决策**：
- 关系运算符 `<` `<=` `>` `>=` 要求两边**类型相同**
- 都是 `number` → 数值比较
- 都是 `string` → UTF-8 字典序比较
- 一边是 `number` 一边是 `string`（或 `null`、`undefined`、`boolean`）→ **`TypeError`**
- `NaN` 与任何值比较都为 `false`（IEEE-754 规定，保留）

```javascript
// GoScript
10 < 9;            // false
"a" < "b";         // true
"10" < "9";        // true （字典序）
"10" < 9;          // TypeError
null < 0;          // TypeError
NaN === NaN;       // true  ← 配合 1.2
NaN < 1;           // false
```

---

## 2. 变量与作用域

### 2.1 默认严格模式

**JS 的问题**：
- 写未声明变量会**静默创建**全局变量（在非严格模式下）。
  ```javascript
  function f() { x = 1; }   // 漏写 var/let
  f();
  console.log(window.x);    // 1 —— 污染全局
  ```
- 严格模式（`"use strict"`）必须**显式**声明。
- 大量老代码从未启用严格模式。

**GoScript 决策**：
- **默认即严格**，无法关闭。
- 写未声明标识符 → 编译期 `ReferenceError`。
- 不存在"脚本模式 vs 模块模式"的分裂。
- `var` 在脚本顶层不再泄漏到全局：
  - 顶层 `let` / `const` → 模块作用域
  - 顶层 `var` → 模块作用域（**不**是全局对象属性）

---

### 2.2 `var` 仅函数作用域（语义不变）

- `var` 声明提升到最近函数/模块顶部，初始值 `undefined`。
- `let` / `const` 块作用域，有 TDZ（暂时性死区）。
- 顶层 `var` 不再挂载到 `globalThis`，需要用 `globalThis.foo` 显式声明（v0.2）。

---

## 3. `this` 绑定

### 3.1 自由函数调用的 `this`

**JS 的问题**：
- 非严格模式：`function f() { console.log(this); } f();` → `globalThis`（浏览器中是 `window`）
- 严格模式：同上 → `undefined`
- 行为取决于是否声明了 `"use strict"`，是个常见的"为什么是 `window`？"调试难题。

**GoScript 决策**：
- 自由函数调用中 `this` 始终是 `undefined`。
- `globalThis` 仍然存在（指向模块作用域），但**不能**通过 `this` 隐式获得。

```javascript
function f() { console.log(this); }
f();                  // undefined
new f();              // 新实例
obj.f();              // obj
(() => this);         // undefined（箭头函数，词法）
```

---

## 4. 对象与数组

### 4.1 `for...in` 只迭代自身可枚举属性

**JS 的问题**：
```javascript
for (let k in {}) console.log(k);  // （非自有属性也能进入）
```
`for...in` 会枚举原型链上所有可枚举的字符串键，
第三方库修改 `Object.prototype` 会污染你的循环。
ESLint 默认开启 `guard-for-in` 规则就是这个原因。

**GoScript 决策**：
- `for...in` 仅迭代**对象自身**的可枚举字符串键。
- 不会进入原型链。
- 数组的 `for...in` 迭代**索引**（字符串形式："0", "1", ...），与 JS 一致。
- 如需遍历原型链，显式 `for (let k in Object.keys(obj))`。

---

### 4.2 数字字面量作对象键不再自动转字符串

**JS 的问题**：
```javascript
let o = { 1: "a" };
o[1];          // "a"   ← 1 被转 "1"
o["01"];       // undefined
```

所有对象键都是字符串（或 Symbol），数字、布尔都会自动转字符串。
对开发者不直观，并且 `{}` 字面量里 `{1: "a"}` 与 `{true: "b"}` 的行为是隐式的。

**GoScript 决策**：
- 对象键依旧是字符串（在哈希表实现层统一）。
- 但 `for...in` 与 `Object.keys` 仍按**插入顺序**输出。
- 数字字面量作 key 时建议显式字符串化 `{"1": "a"}`，以避免歧义。

---

## 5. 循环

### 5.1 `Array.prototype.sort` 默认按数值/字典序

**JS 的问题**：
```javascript
[10, 2, 30].sort();          // [10, 2, 30]   ← 全转字符串字典序
[10, 2, 30].sort((a, b) => a - b);  // [2, 10, 30]  ← 必须显式传比较器
```
默认比较器是字典序，几乎从来不是开发者想要的。

**GoScript 决策**：
- `Array.prototype.sort` **必须**显式传比较器。
- 不传比较器抛 `TypeError`，而不是给一个错误结果。
- 强制开发者思考排序规则。

```javascript
arr.sort();                       // TypeError
arr.sort((a, b) => a - b);        // 数值升序
arr.sort((a, b) => a.localeCompare(b));  // 字符串
```

---

### 5.2 数组 `sort` 不就地修改

**JS 的问题**：`Array.prototype.sort` 是**就地**修改并返回新长度。
`let b = a.sort();` 之后 `a` 与 `b` 指向同一个已排序的数组。
`reverse`、`splice`、`push`、`pop` 等都是这样。
新代码作者会以为 `b = a.sort()` 不动 `a`。

**GoScript 决策**（v0.1 选择保持兼容，v0.2 考虑调整）：
- v0.1 仍为就地修改（与 JS 兼容，便于迁移）。
- 计划在 v0.2 引入 `sorted()`（不修改自身）作为新方法，与 `sort()`（就地）并存。

---

## 6. 数字

### 6.1 `parseInt` 不再吃"奇怪字符"

**JS 的问题**：
```javascript
parseInt("123abc");   // 123  ← 静默截断
parseInt("0x10");     // 16   ← 默认按 hex 解析
parseInt("08");       // 8（严格模式），8（非严格）  ← 老代码兼容性噩梦
```

**GoScript 决策**：
- `parseInt(s, radix?)`：**第一个字符不是数字**就返回 `NaN`。
- `radix` **必须**显式提供（无默认值）。
- 提供 `Number(s)` 作为宽容转换的入口（与 JS 一致）。

```javascript
parseInt("123abc");     // NaN
parseInt("0x10");       // NaN  ← 需要显式 parseInt("0x10", 16)
parseInt("123");        // 123, 但缺少 radix → TypeError
parseInt("123", 10);    // 123
Number("123abc");       // NaN（与 JS 一致）
```

---

### 6.2 安全整数溢出

**JS 的问题**：
```javascript
Number.MAX_SAFE_INTEGER + 1 === Number.MAX_SAFE_INTEGER + 2;  // true ！
```

**GoScript 决策**：
- v0.1 仍为 `float64`，保留此行为。
- 计划在 v0.2 引入 `BigInt`（`123n`）作为 v0.1 数字的"大整数伴侣"。

---

## 7. 危险的/已废弃的特性

| JS 特性 | 状态 | GoScript 替代 |
|---------|------|--------------|
| `with` 语句 | 完全移除 | 显式对象引用 |
| `eval` | 完全移除 | 无（安全考虑） |
| `arguments` 对象 | 完全移除 | `...args` 剩余参数 |
| `switch` / `case` / `default` | 完全移除 | `match` 模式匹配（见 §10） |
| 隐式全局变量 | 移除 | 严格模式默认 |
| 静默失败（failed `JSON.parse`） | 移除 | `JSON.parse` 抛 `SyntaxError` |
| 八进制字面量 `0777` | 移除 | 显式 `0o777` |
| 标签函数（`String.raw\`...\``） | 保留 | 行为不变 |
| `with` / `delete` 复合 | 简化 | `delete` 仅对对象自有可配置属性生效 |
| `void 0` 取代 `undefined` | 没必要 | `undefined` 本身就是字面量 |

---

## 8. 数组构造器

**JS 的问题**：
```javascript
new Array(3);     // [empty × 3]   ← 长度 3，元素全是 hole
new Array(3, 4);  // [3, 4]
new Array(-1);    // RangeError
```

`new Array(n)` 与 `new Array(a, b, ...)` 行为不一致。

**GoScript 决策**：
- 不再支持 `new Array(...)`。
- 用字面量 `[1, 2, 3]` 或 `Array.of(1, 2, 3)`、`Array.from(iterable)`。

---

## 9. 自动分号插入（ASI）

**JS 的问题**：
```javascript
function f() {
  return
  { a: 1 };    // 返回 undefined，因为 ASI 在 return 后插了 ;
}
```

**GoScript 决策**：
- 保留 ASI 机制以兼容 JS 代码风格。
- 但**禁止**换行在以下位置：
  - `return` 与表达式之间
  - `throw` 与表达式之间
  - `break` / `continue` 与标签之间
  - 二元运算符的行首（"leading operator"）
- 不允许的位置直接 `SyntaxError`。
- 工具链（`gofmt` 风格的格式化器）会统一风格。

---

## 10. `switch` → `match` 模式匹配

**JS 的问题**：
- `switch` 使用 `===` 严格相等，但容易写错。
- `case` 默认 fall-through，**必须**显式 `break` —— 漏写就是经典 bug 源。
- `switch` 只能匹配值，不能做范围、类型、解构。
- `switch` 是语句，不是表达式 —— 不能直接 `let x = switch ...`。

**GoScript 决策**：**完全移除 `switch` / `case` / `default`**，引入 `match`：

```javascript
// 基本字面量匹配（无 fall-through）
let name: string = match code {
  200 => "OK",
  301 => "Moved",
  404 => "Not Found",
  500 => "Server Error",
  _   => "Unknown",   // 兜底
};

// OR 模式
match x {
  1 | 2 | 3 => "small",
  _         => "big",
}

// 范围模式
let grade: string = match score {
  0..60     => "F",
  60..70    => "D",
  70..80    => "C",
  80..90    => "B",
  90..=100  => "A",     // 闭区间，含 100
  _         => "invalid",
};

// 绑定 + 守卫
match value {
  n if n < 0  => "negative",
  n if n > 0  => "positive",
  _           => "zero",
}

// 作表达式使用
let msg = match status {
  "ok"    => `Success`,
  "fail"  => `Failed: ${code}`,
};

// 作语句使用
match cmd {
  "quit" => { cleanup(); process.exit(0); },
  "help" => { printHelp(); },
  _      => console.log("unknown"),
}
```

**与 switch 的关键差异**：

| 维度 | `switch` (JS) | `match` (GoScript) |
|------|---------------|--------------------|
| Fall-through | 默认是，必须显式 `break` | **无** —— 每个 arm 独立 |
| 兜底 | `default:` | `_`（wildcard） |
| 范围 | 不支持 | `..` `..=` |
| 多值 | `case 1: case 2:`（依赖 fall-through） | `1 \| 2 \| 3` |
| 守卫 | 不支持 | `if expr` |
| 表达式 | 仅语句 | 表达式与语句均可 |
| 穷尽性检查 | 无 | 缺失 `_` 时编译期警告 |

详细语法见 [`docs/grammar.ebnf`](grammar.ebnf) §1.5、§5；
用法示例见 [`docs/examples/match.gs`](examples/match.gs)。

---

## 11. 仍保留的"JS 风格"行为

为最大化迁移友好性，下列 JS 行为**保留**：

- 数字内部统一为 `float64`（`int` 仅在类型注解层面区分）
- `null` / `undefined` / `0` / `""` / `NaN` / `false` 为 falsy，其他为 truthy
- 字符串模板 `` `${expr}` ``
- 隐式 boxing（基本类型可调用方法：`"abc".toUpperCase()`）
- 数组/字符串是 0 索引、可变长
- `+` 字符串拼接（两边都是 string 时）
- `Date` 与 `Math` 的 API 行为
- 闭包语义（函数捕获外层环境）
- 异步模型（Promise + async/await + 事件循环）

---

## 12. 总结对比表

| 操作 | JavaScript | GoScript |
|------|------------|----------|
| `0 == "0"` | `true` | **SyntaxError** |
| `null === undefined` | `false` | `false` |
| `NaN === NaN` | `false` | **`true`** |
| `typeof null` | `"object"` | **`"null"`** |
| `typeof []` | `"object"` | **`"array"`** |
| `"3" + 1` | `"31"` | **TypeError** |
| `1 + "1"` | `"11"` | **TypeError** |
| `"10" < 9` | `false`（数值） | **TypeError** |
| `function f(){ x=1 }` | 隐式全局 | **ReferenceError** |
| `function f(){ this }` 非严格 | `globalThis` | **`undefined`** |
| `for-in` 迭代原型链 | 是 | **否（仅自身）** |
| `[10,2,30].sort()` | `[10,2,30]` | **TypeError**（要求比较器） |
| `parseInt("123abc")` | `123` | **`NaN`** |
| `with` / `eval` / `arguments` | 支持 | **移除** |
| `switch` | 支持 | **移除 → `match`** |
| 默认严格模式 | 否 | **是** |

---

## 13. 迁移建议

从 JavaScript 迁移到 GoScript 的代码时：

1. 全局替换 `==` → `===`、`!=` → `!==`（建议使用 `jscodeshift` 等 codemod）。
2. 找到所有字符串与数字的 `+`，改为模板字符串。
3. 找到所有隐式全局变量，补上 `let` / `const`。
4. 把 `switch` 改写为 `match`。
5. 打开 `--check-types`，在数组 `sort` / `parseInt` 等位置加显式参数。

> GoScript 的设计目标是：**让你 80% 的 JS 代码可以直接读懂，20% 的脚枪代码在编译期就拒绝。**
