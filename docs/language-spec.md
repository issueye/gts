# GoScript 语言规范

> 本文档描述 GoScript v0.1 实现的语言特性、语义与执行规则。  
> 推荐读者顺序：词法 → 类型 → 表达式 → 语句 → 对象 → 异常 → 模块。

---

## 1. 词法约定

### 1.1 字符集

- 源文件为 UTF-8 编码。
- 行终止符：`\n`、`\r`、`\r\n`。
- 空白符（空格、制表符、垂直制表符、换页符）仅在分隔 token 时有意义。

### 1.2 注释

```javascript
// 单行注释，直到行尾

/*
   块注释，可跨行
   /* 支持嵌套 */
*/
```

### 1.3 标识符

```
identifier  :=  (letter | "_" | "$")  (letter | digit | "_" | "$")*
```

- 大小写敏感。
- 标识符字符支持 Unicode（依据 `unicode.IsLetter` / `IsDigit`）。
- `$` 与 `_` 是合法起始字符（与 JS 一致）。

### 1.4 关键字（保留字）

```
break        catch       class       const       continue
delete       else        export      extends     false
finally      for         function    if          import
in           instanceof  let         match       new
null         return      super       this        throw
true         try         typeof      undefined   var
void         while       async       await       of
static       as          from
```

**保留但 v0.1 移除**（不能再作标识符，且这些功能不存在）：
`with` `eval`（内置）`arguments` `switch` `case` `default` `do`

> 不再保留的"伪关键字"列表见 [`docs/bad-parts-fixed.md`](bad-parts-fixed.md) §10。

### 1.5 字面量

#### 1.5.1 数字

```
123        // 整数
1.5        // 浮点
1e3        // 指数
0x1F       // 十六进制
0b1010     // 二进制
0o17       // 八进制
```

> 内部统一以 `float64` 存储。区分 `int` 仅靠类型注解。

#### 1.5.2 字符串

```javascript
"hello"
'single quote'
"包含 \"转义\" 和 \n 换行"
"Unicode: \u{1F600}"
```

支持的转义：`\n \t \r \\ \' \" \` \0 \b \f \xHH \uHHHH \u{H...H}`

#### 1.5.3 模板字符串

```javascript
let name = "Alice";
let msg = `Hello, ${name}!`;
let calc = `1 + 1 = ${1 + 1}`;
```

模板字符串支持任意深度嵌套 `${...}`，可换行。

#### 1.5.4 布尔、null、undefined

```javascript
true  false  null  undefined
```

> `null` 与 `undefined` 是不同值。`null == undefined` 为 `true`（双等），`null === undefined` 为 `false`（三等）。

---

## 2. 类型系统

### 2.1 运行时类型

| 类型 | 字面量 | 备注 |
|------|--------|------|
| `number` | `1`, `1.5`, `NaN`, `Infinity` | 内部 `float64` |
| `string` | `"a"`, `'b'`, `` `c` `` | 不可变 UTF-8 |
| `boolean` | `true` / `false` | — |
| `null` | `null` | 显式空 |
| `undefined` | `undefined` | 未初始化 / 缺省 |
| `object` | `{...}` | 键值集合，原型链根 |
| `array` | `[...]` | 可变长、带 `length` |
| `function` | `function() {}` | 一等公民 |
| `class` | `class C {}` | v0.1 中类也是对象 |
| `instance` | `new C()` | 用户实例 |
| `promise` | `new Promise(...)` | 异步值 |
| `error` | `new Error("...")` | 异常对象 |

> 运行时通过 `typeof` 可获得以下字符串之一：
> `"number"` `"string"` `"boolean"` `"null"` `"undefined"`
> `"object"` `"array"` `"function"` `"class"` `"promise"`
> `"error"`（v0.2） `"symbol"`（v0.2）
>
> **与 JS 的关键差异**：
> - `typeof null` → `"null"`（不是 `"object"`）
> - `typeof []` → `"array"`（不是 `"object"`）
> - `typeof new Promise(...)` → `"promise"`
> - `typeof class C{}` → `"class"`

### 2.2 可选类型注解

```javascript
let x: number = 42;
const name: string = "GoScript";
let arr: Array<number> = [1, 2, 3];
let maybe: number | null = null;

function add(a: number, b: number): number {
  return a + b;
}

class User {
  name: string;
  age: int;
  constructor(name: string, age: int) {
    this.name = name;
    this.age = age;
  }
  greet(): string {
    return `Hi, I'm ${this.name}`;
  }
}
```

支持的类型表达式（见 EBNF `type-expression`）：

| 形态 | 含义 |
|------|------|
| `number` `string` `boolean` `null` `undefined` `any` `void` | 基本类型 |
| `int` `float` | 数字的细化（运行时检查） |
| `T[]` 或 `Array<T>` | 数组 |
| `T \| U` | 联合类型 |
| `T?` | 等价于 `T \| null` |
| `{ key: T, key2?: T2 }` | 对象结构类型 |
| `(a: T, b: T) => U` | 函数类型 |
| 标识符（用户类） | 类类型（结构兼容） |

#### 规则

1. **未标注 = 不检查**，不破坏动态特性。
2. **标注必须匹配**（除非用 `any`）。
3. 类型错误抛 `TypeError`，不静默。
4. 数组/对象结构类型使用**子类型结构匹配**，不需要 `===`。
5. 联合类型运行时按"最具体 → 最一般"匹配；任一分支匹配即通过。

> 启用方式：命令行 `gs --check-types script.gs` 或 API `Evaluator{TypeCheck: true}`。

---

## 3. 变量与作用域

### 3.1 三种声明

```javascript
let x = 1;        // 块作用域，可重赋值
const y = 2;      // 块作用域，不可重赋值
var z = 3;        // 函数作用域，声明提升
```

- `let` / `const` 绑定在最近的块（`{...}`）作用域。
- `const` 绑定不可再赋值，但**对象/数组内容可变**（浅不可变）。
- `var` 绑定提升到最近函数 / 全局顶部；值为 `undefined` 直到赋值。

### 3.2 作用域链

```javascript
let a = 1;
{
  let a = 2;
  console.log(a);  // 2
}
console.log(a);    // 1
```

块作用域在每次进入时创建新 `Environment`，退出时丢弃。

### 3.3 闭包

```javascript
function makeCounter() {
  let n = 0;
  return function() {
    n = n + 1;
    return n;
  };
}

const c = makeCounter();
c(); // 1
c(); // 2
```

> 闭包通过**环境指针**实现，被捕获的变量在外层函数返回后仍然存活。

---

## 4. 表达式

### 4.1 字面量与变量

```javascript
42
"hello"
true
null
undefined
x
```

### 4.2 运算符

#### 4.2.1 算术

| 运算符 | 说明 |
|--------|------|
| `+` | **严格**：两边必须是相同类型（number+number 或 string+string），其他组合抛 `TypeError` |
| `-` | 减；操作数通过 `Number(x)` 隐式转换 |
| `*` | 乘；同上 |
| `/` | 除；同上 |
| `%` | 取模；同上 |
| `**` | 幂（右结合）；同上 |
| `++` `--` | 前/后自增自减（仅对 `number`） |

**关于 `+` 的严格语义**：

```javascript
1 + 2;            // 3
"a" + "b";        // "ab"
1 + "1";          // TypeError
"x" + null;       // TypeError
`${1}1`;          // "11"  ← 显式拼接
String(1) + "1";  // "11"  ← 显式转换
```

> 详见 [`docs/bad-parts-fixed.md`](bad-parts-fixed.md) §1.5。

#### 4.2.2 相等与比较

> **注意**：`==` 与 `!=` 在 GoScript 中是**语法错误**。本节只描述 `===` / `!==` 和关系运算符。

| 运算符 | 说明 |
|--------|------|
| `===` | 严格相等：类型 + 值都相同 |
| `!==` | 严格不等 |
| `<` `<=` `>` `>=` | 关系：要求两边**类型相同**（都是 number 或都是 string） |

**严格相等规则**：

| 表达式 | 结果 |
|--------|------|
| `1 === 1` | `true` |
| `1 === "1"` | `false`（类型不同） |
| `null === undefined` | `false` |
| `null === null` | `true` |
| `undefined === undefined` | `true` |
| `NaN === NaN` | **`true`**（与 JS 不同） |
| `{} === {}` | `false`（按引用） |
| `[] === []` | `false` |
| `[1] === [1]` | `false` |

**关系运算符规则**：

| 表达式 | 结果 |
|--------|------|
| `1 < 2` | `true` |
| `"a" < "b"` | `true`（UTF-8 字典序） |
| `"10" < "9"` | `true`（字典序，非数值！） |
| `"10" < 9` | **TypeError** |
| `null < 0` | **TypeError** |
| `NaN < 1` | `false`（IEEE-754） |

> 详见 [`docs/bad-parts-fixed.md`](bad-parts-fixed.md) §1.3、§1.6。

#### 4.2.3 逻辑

| 运算符 | 行为 |
|--------|------|
| `&&` | 左为真返回右，否则返回左 |
| `\|\|` | 左为假返回右，否则返回左 |
| `??` | 左为 `null` / `undefined` 返回右，否则返回左 |
| `!` | 取反（先转布尔） |

> 逻辑与/或/`??` **不**返回 `true/false`，而是返回对应的**操作数值**。

#### 4.2.4 位运算

| 运算符 | 说明 |
|--------|------|
| `&` `\|` `^` `~` | 与、或、异或、取反（先把数字转为 int32） |
| `<<` `>>` `>>>` | 移位 |

> 位运算会先把操作数转为 int32，溢出按补码处理。

#### 4.2.5 赋值

```
=  +=  -=  *=  /=  %=  **=  <<=  >>=  >>>=  &=  |=  ^=
```

> 复合赋值运算符**不**做类型转换：
> `let s: string = "1"; s += 1;` → **TypeError**（因为 `s` 是 string，右侧 `1` 是 number）

#### 4.2.6 其它

- `typeof` 表达式，返回类型字符串（见 §2.1）。
- `void expr` 求值后返回 `undefined`。
- `delete obj.key` 移除属性（仅对 `object` 自身的可配置属性），返回 `boolean`。
- `await expr` 等待 `Promise`（详见 [`docs/async-model.md`](async-model.md)）。
- `in` 判断键是否在对象中（**仅自身属性**，不含原型链）：`"toString" in {}` → `false`。
- `instanceof` 判断原型链：`{} instanceof Object` → `true`。

### 4.3 条件

```javascript
let r = a > 0 ? "positive" : "non-positive";
```

### 4.4 成员与调用

```javascript
obj.prop
obj["prop"]
arr[0]
fn(arg1, arg2)
new Date()
obj?.prop          // 可选链，null/undefined 时短路
obj?.method?.()
arr?.[0]
fn?.()
```

- 调用 `f()` 时，`this` 绑定到 `f` 之前的"基"：
  - `obj.f()` → `this = obj`
  - `f()` → `this = undefined`（**始终**，不依赖严格模式）
  - `new f()` → `this = 新实例`
- 箭头函数 `this` 词法绑定到定义时的外层 `this`。

### 4.5 数组字面量

```javascript
[1, 2, 3]
[1, , 3]           // hole
[1, ...other, 5]   // 展开
```

### 4.6 对象字面量

```javascript
let x = 1, y = 2;
let p = { x, y, name: "p" };                  // 简写
let m = { value: 1, ["k_" + 1]: 2 };          // 计算键
let a = { ...other, b: 3 };                   // 展开
let o = { greet() { return "hi"; } };        // 方法简写
```

### 4.7 函数表达式

```javascript
let f = function add(a, b) { return a + b; };
let g = function(a, b) { return a + b; };
let h = (a, b) => a + b;
let i = a => a * 2;
let j = () => 42;
let k = async (x) => { return await x; };
```

箭头函数体是表达式时为隐式返回；是块时需显式 `return`。

### 4.8 模板字符串

```javascript
let s = `value = ${compute()}, ts = ${Date.now()}`;
```

### 4.9 `match` 表达式

`match` 是 GoScript 的**模式匹配**机制（替代 JS 的 `switch`），
并且它**首先是表达式**，可以返回值，也可以作为语句。

```javascript
// 作为表达式
let name: string = match code {
  200 => "OK",
  404 => "Not Found",
  _   => "Unknown",
};

// 作为语句（体用块）
match cmd {
  "quit" => { cleanup(); process.exit(0); },
  "help" => { printHelp(); },
  _      => console.log("unknown"),
}
```

完整语法与模式见 §5.4。

---

## 5. 语句

### 5.1 块

```javascript
{
  let x = 1;
  console.log(x);
}
```

### 5.2 条件

```javascript
if (a > 0) {
  // ...
} else if (a === 0) {
  // ...
} else {
  // ...
}
```

### 5.3 循环

```javascript
while (cond) { /* ... */ }

for (let i = 0; i < n; i++) { /* ... */ }
for (let i in arr) { /* i 是索引/键 */ }
for (let v of arr) { /* v 是值 */ }

for (let k in obj) { /* k 是键 */ }
for (let v of Object.values(obj)) { /* v 是值 */ }
```

- `for-in` 遍历数组时迭代**索引**（字符串形式 `"0"`、`"1"`、...），
  遍历对象时迭代**自身可枚举字符串键**（**不**含原型链）。
- `for-of` 要求右侧实现迭代协议；v0.1 仅 `Array` 与内置 `String` 适配。
- `break` / `continue` 可选带标签 `break outer;`。

> `for-in` 不迭代原型链，这是与 JS 的重要差异 —— 详见 [`docs/bad-parts-fixed.md`](bad-parts-fixed.md) §4.1。

### 5.4 `match` 模式匹配（替代 `switch`）

> GoScript **不支持** `switch` / `case` / `default`。
> 改用 `match`，功能更强大且无 `fall-through` 陷阱。

#### 5.4.1 语法

```
match expr {
  pattern [if guard] => body,
  pattern [if guard] => body,
  ...
}
```

- `body` 可以是**表达式**或**块语句**。
- 当 `match` 用作表达式时，所有 arm body 都应是表达式（返回该值）。
- 当 `match` 用作语句时，arm body 可以是块。
- arm 之间以 `,` 分隔（最后一条 `,` 可省略）。
- **没有** `fall-through`；每个 arm 独立匹配并立即结束 `match`。

#### 5.4.2 模式

| 模式 | 含义 | 示例 |
|------|------|------|
| 字面量 | 与该值 `===` 匹配 | `1`, `"hi"`, `true`, `null` |
| 标识符 | 绑定：把被匹配值绑定到该名字 | `n`（绑定到 `n`） |
| `_` | 通配：匹配任意值，不绑定 | `_` |
| OR | 任一子模式匹配即可 | `1 \| 2 \| 3` |
| 范围 | 闭区间或半开区间 | `1..10`（不含 10）、`1..=100`（含 100） |
| 守卫 | 模式匹配后额外条件判断 | `n if n > 0` |

#### 5.4.3 示例

```javascript
// 1) HTTP 状态码
let label: string = match status {
  200 => "OK",
  301 => "Moved",
  404 => "Not Found",
  500..599 => "Server Error",
  _ => "Unknown",
};

// 2) OR 模式 + 范围 + 守卫
match x {
  1 | 2 | 3 => console.log("small"),
  n if n < 0 => console.log("negative"),
  n if n > 100 => console.log("big"),
  10..=20 => console.log("10-20"),
  _ => console.log("other"),
}

// 3) 用作语句（块体）
match cmd {
  "quit" => { cleanup(); process.exit(0); },
  "help" => { printHelp(); },
  _      => { console.log("unknown:", cmd); },
}

// 4) 嵌套 match
match shape {
  "circle"    => match radius { 0 => "point", _ => "disc" },
  "rectangle" => "rect",
  _           => "unknown shape",
}
```

#### 5.4.4 规则

1. **穷尽性**：缺失兜底（`_`）时编译器可发出警告（默认警告，不阻断）。
2. **无匹配**：运行时若无 arm 匹配且无 `_` 兜底，抛 `MatchError`。
3. **绑定作用域**：arm 体可见该 arm 内所有 `identifier-pattern` 与守卫引入的变量。
4. **绑定与字面量冲突**：标识符模式按"绑定"处理，不会与字面量冲突；若想强制与某常量值比较，可加守卫：
   ```javascript
   match v {
     0 => "zero",
     n if n === null => "null",   // 因 null 不是字面量模式
     n if n === undefined => "undef",
     _ => "other",
   }
   ```
5. **类型注解**：`match` 表达式可以标注整体类型：
   ```javascript
   let x: string = match code { 200 => "OK", _ => "?" };
   ```

#### 5.4.5 模式（v0.2 规划）

- 数组模式 `[1, 2, x, ...rest]`
- 对象模式 `{ name, age: n }`
- 类型测试 `n is string`
- 嵌套模式

### 5.5 返回 / 跳转

```javascript
return value;     // 函数返回值（无值则返回 undefined）
return;           // 等价 return undefined;
break;
continue;
```

### 5.6 异常

```javascript
try {
  mayThrow();
} catch (e: Error) {
  console.error(e.message);
} finally {
  cleanup();
}

throw new Error("oops");
throw "string error";   // 也允许，建议抛 Error
```

- `catch` 形参为 `Error` 子类型时按结构匹配；形参无注解则匹配任意。
- `finally` 在 `try` / `catch` 之后**总是**执行。
- `finally` 中的 `return` 会覆盖 `try` / `catch` 的返回值。

### 5.7 标签

```javascript
outer: for (let i = 0; i < n; i++) {
  for (let j = 0; j < m; j++) {
    if (a[i][j] === target) break outer;
  }
}
```

---

## 6. 函数

### 6.1 声明与表达式

```javascript
function add(a, b) { return a + b; }

const sub = function(a, b) { return a - b; };
const mul = (a, b) => a * b;
```

### 6.2 默认参数

```javascript
function greet(name = "World") {
  return "Hello, " + name;
}
```

### 6.3 剩余参数

```javascript
function sum(...nums) {
  return nums.reduce((a, b) => a + b, 0);
}
```

### 6.4 参数解构（v0.1 语法占位）

```javascript
function f({x, y}, [a, b]) { /* ... */ }
```

> 解构在 v0.2 评估。

### 6.5 `this` 绑定

| 调用形态 | `this` |
|---------|--------|
| `f()` | `undefined`（顶层） / 全局对象 |
| `obj.f()` | `obj` |
| `arr[i]()` | `arr[i]` 取出时的基 |
| `new f()` | 新建实例 |
| 箭头函数 | 定义时词法绑定的 `this` |
| `bind(thisArg)` | `thisArg`（v0.1 暂不实现） |
| `call(thisArg, ...)` | `thisArg`（v0.1 暂不实现） |

---

## 7. 数组

```javascript
let a = [1, 2, 3];
a.push(4);          // [1,2,3,4]
a.pop();            // 4
a.length;           // 3
a[0] = 10;
for (let v of a) console.log(v);
```

详见 [`docs/builtins.md`](builtins.md)。

---

## 8. 对象

### 8.1 创建

```javascript
let o = { a: 1, b: 2 };
o.c = 3;
delete o.a;
"a" in o;           // false
"b" in o;           // true
```

### 8.2 原型

```javascript
const proto = { greet() { return "hi"; } };
const obj = Object.create(proto);
obj.name = "x";
obj.greet();        // "hi"  —— 来自原型
```

### 8.3 枚举属性顺序

字符串键按插入顺序、整数键按升序、Symbol 不支持。

---

## 9. 类

### 9.1 基本类

```javascript
class Animal {
  name: string;

  constructor(name: string) {
    this.name = name;
  }

  speak(): string {
    return `${this.name} makes a sound`;
  }
}
```

### 9.2 继承

```javascript
class Dog extends Animal {
  breed: string;

  constructor(name: string, breed: string) {
    super(name);
    this.breed = breed;
  }

  speak(): string {
    return `${this.name} barks`;
  }
}
```

- 派生类 `constructor` **必须**调用 `super()` 才能使用 `this`。
- 方法定义在 `class.prototype` 上（实例的原型链对象）。
- 字段在 `constructor` 顶部初始化（按声明顺序）。

### 9.3 静态成员

```javascript
class MathUtil {
  static PI = 3.14159;
  static square(x) { return x * x; }
}

MathUtil.PI;
MathUtil.square(3);  // 9
```

### 9.4 Getter / Setter（v0.1 暂不实现）

---

## 10. 异步

### 10.1 Promise

```javascript
let p = new Promise((resolve, reject) => {
  setTimeout(() => resolve(42), 1000);
});

p.then(v => console.log(v))
 .catch(e => console.error(e));
```

### 10.2 async / await

```javascript
async function fetchUser(id) {
  let resp = await fetch(`/api/users/${id}`);
  if (!resp.ok) throw new Error("HTTP " + resp.status);
  return await resp.json();
}

(async () => {
  try {
    let u = await fetchUser(1);
    console.log(u);
  } catch (e) {
    console.error(e);
  }
})();
```

详见 [`docs/async-model.md`](async-model.md)。

---

## 11. 模块

```javascript
// math.gs
export function add(a, b) { return a + b; }
export const PI = 3.14159;
export default class Vector { /* ... */ }
```

```javascript
// app.gs
import Vector, { add, PI } from "./math.gs";
```

- 模块解析：相对路径基于当前文件所在目录。
- 文件级缓存：每个文件只执行一次。
- 顶层 `await` v0.1 不支持。

> 详细模块规范在 v0.2 完善。

---

## 12. 错误对象

```go
new Error("msg")
new TypeError("type wrong")
new RangeError("out of range")
new ReferenceError("undefined var")
new SyntaxError("bad syntax")  // 运行时抛，用于动态 eval 等场景
```

字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `message` | `string` | 人类可读信息 |
| `name` | `string` | 类型名，默认 `"Error"` |
| `stack` | `string` | 堆栈，格式见下 |

堆栈格式：

```
Error: msg
    at fnName (file:line:col)
    at <main> (file:line:col)
```

---

## 13. 严格性与保留差异

> 本节是**速览**。完整说明见 [`docs/bad-parts-fixed.md`](bad-parts-fixed.md)。

GoScript 相对 JavaScript 的**有意差异**可分三类：

### 13.1 修复类型比较的"经典坑"

| 项 | JS | GoScript |
|----|----|----------|
| `==` / `!=` | 抽象相等 | **语法错误** |
| `===` 数字 | `NaN === NaN` 为 `false` | `NaN === NaN` 为 `true` |
| `typeof null` | `"object"` | `"null"` |
| `typeof []` | `"object"` | `"array"` |
| `+` 混合类型 | 字符串拼接 | `TypeError` |
| `<` `>` 混合类型 | 隐式数值转换 | `TypeError` |
| `null === undefined` | `false` | `false`（无 `==` 干扰） |

### 13.2 修复作用域 / `this` / 隐式全局的坑

| 项 | JS | GoScript |
|----|----|----------|
| 隐式全局变量 | 非严格模式下静默创建 | **默认严格**，未声明变量 → `ReferenceError` |
| 顶层 `var` 污染全局 | 挂到 `globalThis` | 仅模块作用域 |
| 自由函数 `this` | 非严格 = `globalThis` | **始终** `undefined` |
| `for-in` 迭代原型链 | 是 | **否**（仅自身） |
| `Array.prototype.sort` 默认 | 字典序（几乎从不想要） | **TypeError**（必须显式比较器） |
| `parseInt` 静默截断 | `"123abc" → 123` | `NaN` |
| `parseInt` 默认 radix | 10/8 视模式而定 | **必须**显式 radix |
| `new Array(3)` | 创建稀疏数组 | **移除**（用 `Array.of` / 字面量） |

### 13.3 移除的危险/废弃特性

| 项 | 替代 |
|----|------|
| `with` 语句 | 移除 |
| `eval` | 移除（安全考虑） |
| `arguments` 对象 | `...args` 剩余参数 |
| `switch` / `case` / `default` | **`match` 模式匹配**（见 §5.4） |
| `do...while` | 暂不实现 |
| 八进制字面量 `0777` | 显式 `0o777` |

---

## 14. 兼容性矩阵

| 特性 | v0.1 | 备注 |
|------|------|------|
| 基本类型、运算符 | ✅ | 严格 `+`、严格比较、移除 `==` |
| 控制流 | ✅ | if/while/for/for-in/for-of/match |
| 模式匹配 `match` | ✅ | 字面量/OR/范围/守卫/绑定 |
| 函数 / 闭包 | ✅ | 箭头函数 `this` 词法 |
| 数组 / 对象 | ✅ | `for-in` 仅自身 |
| 解构赋值 | ❌ | v0.2（含在 match 模式中） |
| 模板字符串 | ✅ | |
| 类 / 继承 | ✅ | 不支持 getter/setter、私有字段 `#` |
| async / await | ✅ | |
| Promise | ✅ | |
| try / catch / finally | ✅ | |
| import / export | ⚠️ 语法 | 运行时按文件边界 |
| 类型注解 | ✅ | 需开启 `--check-types` |
| `==` / `!=` | ❌ 语法错误 | 使用 `===` / `!==` |
| `switch` | ❌ | 使用 `match` |
| `with` / `eval` / `arguments` | ❌ | — |
| Symbol | ❌ | v0.2 |
| Iterator / Generator | ⚠️ for-of 数组 | 生成器函数 v0.2 |
| Proxy / Reflect | ❌ | 远期 |

---

> 接下来：[`docs/bad-parts-fixed.md`](bad-parts-fixed.md) 了解被修复的 JS 缺点细节；
> 或 [`docs/async-model.md`](async-model.md) 深入异步运行时。
