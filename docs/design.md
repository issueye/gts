# GoScript 架构与详细设计

> 版本：v0.1 设计文档。  
> 目标：解释器可工作的最小可行实现（MVP），覆盖 JS 核心 + `async/await` + 可选类型注解。
>
> 注意：本文保留了部分目标架构描述，例如 Resolver、TypeChecker 和公开嵌入 API。当前实现状态请先看 [`project-analysis.md`](project-analysis.md)，开发计划请看 [`development-plan.md`](development-plan.md)。

---

## 1. 设计目标与非目标

### 1.1 设计目标

1. **可读性优先**：源码即文档，模块边界清晰，命名一致。
2. **JS 习惯转移成本低**：熟悉 JavaScript 的开发者几乎零成本上手。
3. **可嵌入**：核心组件以 Go 包形式暴露，允许在 Go 应用中执行脚本片段。
4. **错误友好**：精确的行列号、可读的错误信息、堆栈追踪。
5. **教学价值**：体现"现代动态语言解释器"的关键工程点（闭包、异步、对象模型）。

### 1.2 非目标（v0.1）

- 不做 JIT，不做字节码 VM（明确放弃以换取实现速度与代码清晰度）。
- 不做浏览器集成，不做 DOM。
- 不做 ES Module 的全静态分析（仅运行期导入）。
- 不做完整的 ECMAScript 规范兼容（仅子集）。
- 不做并发多线程（事件循环单线程，宿主线程由 Go 调度器决定）。

### 1.3 设计哲学：修复 JS 的"Bad Parts"

> **保留 JS 的语法熟悉度，消除那些连老手也会踩的坑。**  
> 详细列表见 [`docs/bad-parts-fixed.md`](bad-parts-fixed.md)；本章只列影响架构的关键决策。

| 决策 | 影响 |
|------|------|
| 移除 `==` / `!=` | 词法层直接拒绝，简化求值器 |
| `NaN === NaN` 为 `true` | 求值器对 `===` 不再走 IEEE-754 NaN 特殊路径 |
| `typeof null` / `typeof []` 返回独立字符串 | `Type()` 方法直接返回精确字符串 |
| `+` 严格（混合类型 TypeError） | 求值器在 `evalNumberInfix` / `evalStringInfix` 之外抛错 |
| 比较运算符类型必须一致 | 在 `evalRelational` 入口校验类型 |
| 默认严格模式 | Resolver 阶段对未声明写入抛 `ReferenceError` |
| 自由函数 `this = undefined` | 不再注入 `globalThis` 作为 `this` |
| `for-in` 不含原型链 | `Object.Keys` 替代 `for-in` 实现 |
| `Array.sort` 必须比较器 | 不传 `cmp` 直接抛 `TypeError` |
| `parseInt` 严格化 | 首字符非数字 → `NaN`；必须显式 radix |
| 移除 `with` / `eval` / `arguments` | 词法层不识别，AST 不包含这些节点 |
| 移除 `switch` / `case` / `default` | 由 `match` 模式匹配替代（见 §4.3.3） |

---

## 2. 总体架构

```
┌────────────────────────────────────────────────────────────┐
│                      源码（.gs）                          │
└────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌────────────────────────────────────────────────────────────┐
│  Lexer        ── 字符流 → Token 流                          │
│  (internal/lexer)                                          │
└────────────────────────────────────────────────────────────┘
                            │ []Token
                            ▼
┌────────────────────────────────────────────────────────────┐
│  Parser       ── Pratt 表达式 + 递归下降语句 → AST          │
│  (internal/parser)                                         │
└────────────────────────────────────────────────────────────┘
                            │ *ast.Program
                            ▼
┌────────────────────────────────────────────────────────────┐
│  Resolver     ── 作用域分析、闭包变量捕获、this 绑定         │
│  (internal/resolver)                                       │
└────────────────────────────────────────────────────────────┘
                            │ 已解析的 AST
                            ▼
┌────────────────────────────────────────────────────────────┐
│  TypeChecker  ── 可选类型注解校验（仅检查显式声明的）         │
│  (internal/typechecker)                                    │
└────────────────────────────────────────────────────────────┘
                            │ 类型已校验的 AST
                            ▼
┌────────────────────────────────────────────────────────────┐
│  Evaluator    ── 树遍历求值 + 环境链 + 控制流                │
│  (internal/evaluator)                                      │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Async Runtime   ── Promise / 微任务 / await 续延    │  │
│  │ (internal/async)                                     │  │
│  └──────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ StdLib           ── console / Math / JSON / Object    │  │
│  │ (internal/stdlib)                                    │  │
│  └──────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────┘
                            │
                            ▼
                     脚本输出 / 异常
```

### 2.1 阶段划分

| 阶段 | 输入 | 输出 | 关键算法 |
|------|------|------|---------|
| 词法分析 | 字符串 | `[]Token` | 手写状态机（DFA 风格） |
| 语法分析 | `[]Token` | `*ast.Program` | Pratt 表达式 + 递归下降语句 |
| 解析 | `*ast.Program` | 带作用域信息的 AST | 两次遍历，识别 free 变量 |
| 类型检查 | 带作用域的 AST | 检查后的 AST | 基于声明的对偶式校验 |
| 求值 | AST | 副作用 | 树遍历 + 环境链 + 异常栈 |

---

## 3. 词法分析（Lexer）

### 3.1 词法单元（Token）

```go
type TokenType int

const (
    // 字面量
    TOKEN_NUMBER TOKEN_TYPE = iota
    TOKEN_STRING
    TOKEN_TEMPLATE
    TOKEN_IDENT
    // 关键字
    TOKEN_LET TOKEN_CONST TOKEN_VAR
    TOKEN_FUNCTION TOKEN_CLASS TOKEN_EXTENDS
    TOKEN_IF TOKEN_ELSE TOKEN_WHILE TOKEN_FOR TOKEN_IN TOKEN_OF
    TOKEN_RETURN TOKEN_BREAK TOKEN_CONTINUE
    TOKEN_TRUE TOKEN_FALSE TOKEN_NULL TOKEN_UNDEFINED
    TOKEN_NEW TOKEN_THIS TOKEN_SUPER
    TOKEN_TRY TOKEN_CATCH TOKEN_FINALLY TOKEN_THROW
    TOKEN_ASYNC TOKEN_AWAIT
    TOKEN_IMPORT TOKEN_EXPORT TOKEN_FROM TOKEN_AS
    // 运算符
    TOKEN_PLUS TOKEN_MINUS TOKEN_STAR TOKEN_SLASH TOKEN_PERCENT
    TOKEN_EQ TOKEN_EQ_EQ_EQ TOKEN_NEQ_EQ
    (* 注：== 与 != 已被移除 —— 见 docs/bad-parts-fixed.md §1.1 *)
    TOKEN_LT TOKEN_LT_EQ TOKEN_GT TOKEN_GT_EQ
    TOKEN_AND_AND TOKEN_OR_OR TOKEN_BANG
    TOKEN_AMP TOKEN_PIPE TOKEN_CARET TOKEN_TILDE TOKEN_LT_LT TOKEN_GT_GT TOKEN_GT_GT_GT
    TOKEN_AMP_EQ TOKEN_PIPE_EQ TOKEN_CARET_EQ TOKEN_LT_LT_EQ TOKEN_GT_GT_EQ TOKEN_GT_GT_GT_EQ TOKEN_PLUS_EQ TOKEN_MINUS_EQ ...
    TOKEN_PLUS_PLUS TOKEN_MINUS_MINUS
    TOKEN_ARROW            // =>
    TOKEN_ELLIPSIS         // ...
    TOKEN_QUESTION         // ?
    // 分隔符
    TOKEN_LPAREN TOKEN_RPAREN TOKEN_LBRACE TOKEN_RBRACE TOKEN_LBRACK TOKEN_RBRACK
    TOKEN_COMMA TOKEN_SEMI TOKEN_COLON TOKEN_DOT
    // 特殊
    TOKEN_EOF TOKEN_ILLEGAL
)
```

### 3.2 关键规则

- **数字字面量**：支持十进制 `123`、浮点 `1.5`、指数 `1e3`、十六进制 `0x1F`、二进制 `0b1010`。
  内部统一存为 `float64`（与 JS 一致）。
- **字符串字面量**：单引号 / 双引号。支持常见转义 `\n \t \r \\ \" \' \0 \xHH \u{HHHH}`。
- **模板字符串**：反引号。支持 `${expr}` 插值。词法层识别为独立 token，解析层拆解。
- **标识符**：以字母 / `_` / `$` 开头，后续字符可含数字。允许 Unicode 标识符。
- **关键字**：保留字集合通过 `keywords` map 转换。
- **注释**：`// line` 与 `/* block */`，词法层直接跳过。
- **行号 / 列号**：每个 token 携带 `Line`、`Column`、`Offset`，用于错误信息。
- **数值越界**：词法阶段不报错，由求值阶段抛 `RangeError`。

### 3.3 错误恢复

词法错误使用 `lexer.Error()` 收集到错误列表中，**不立即 panic**。  
`Parser` 在遇到 `TOKEN_ILLEGAL` 时继续尝试同步（同步到下一个语句边界），保证一次性报告多条错误。

---

## 4. 语法分析（Parser）

### 4.1 总体策略

- **Pratt 解析器**处理表达式（运算符优先级 + 中缀 / 前缀 / 后缀）。
- **递归下降**处理语句与声明。
- **前瞻 1~2 个 token** 解决冲突。

### 4.2 优先级表（从低到高）

| 优先级 | 运算符 | 结合性 |
|--------|--------|--------|
| 1 | `,` | 左 |
| 2 | `=` `+=` `-=` `*=` `/=` `%=` `<<=` `>>=` `>>>=` `&=` `\|=` `^=` | 右 |
| 3 | `?:` | 右 |
| 4 | `\|\|` | 左 |
| 5 | `&&` | 左 |
| 6 | `\|` | 左 |
| 7 | `^` | 左 |
| 8 | `&` | 左 |
| 9 | `===` `!==` | 左 |（`==` / `!=` 已移除） |
| 10 | `<` `<=` `>` `>=` `in` `instanceof` | 左 |
| 11 | `<<` `>>` `>>>` | 左 |
| 12 | `+` `-` | 左 |
| 13 | `*` `/` `%` | 左 |
| 14 | `**` | 右 |
| 15 | 一元 `!` `-` `+` `~` `typeof` `void` `delete` `await` | 右 |
| 16 | 后缀 `++` `--` | — |
| 17 | `()` `[]` `.` | 左 |
| 18 | 字面量 / 标识符 / 模板 | — |

> 详见 [`docs/grammar.ebnf`](grammar.ebnf)。

### 4.3 AST 节点

```go
type Node interface {
    TokenLiteral() string
    Pos() Position
}

type Program struct {
    Body      []Statement
    SourceFile string
    Imports   []*ImportDecl
    Exports   map[string]bool
}

type Statement interface {
    Node
    statementNode()
}

type Expression interface {
    Node
    expressionNode()
}
```

#### 4.3.1 表达式节点

| 节点 | 关键字段 |
|------|---------|
| `Identifier` | `Name`, `Resolved Scope?` |
| `NumberLiteral` | `Value float64`, `Raw string`, `IsInt bool` |
| `StringLiteral` | `Value string` |
| `TemplateLiteral` | `Parts []string`, `Exprs []Expression` |
| `BooleanLiteral` | `Value bool` |
| `NullLiteral` | — |
| `ArrayLiteral` | `Elements []Expression` |
| `ObjectLiteral` | `Properties []Property`（key/value/shorthand/spread） |
| `PrefixExpr` | `Op string`, `Right Expression` |
| `InfixExpr` | `Op string`, `Left, Right Expression` |
| `TernaryExpr` | `Cond, Consequent, Alternate Expression` |
| `AssignmentExpr` | `Op string`, `Left Expression`, `Right Expression` |
| `CallExpr` | `Callee Expression`, `Args []Expression` |
| `MemberExpr` | `Object Expression`, `Property Expression`, `Computed bool` |
| `IndexExpr` | `Left, Index Expression` |
| `FunctionLiteral` | `Name string`, `Params []Parameter`, `Body *BlockStmt`, `IsAsync bool`, `TypeAnnotation *FuncType?` |
| `ArrowFunctionLiteral` | `Params []Parameter`, `Body Expression|*BlockStmt`, `IsAsync bool` |
| `NewExpr` | `Callee Expression`, `Args []Expression` |
| `ClassLiteral` | `Name string`, `SuperClass Expression?`, `Body *ClassBody`, `TypeParams` |
| `ThisExpr` | — |
| `SuperExpr` | `Method string` |
| `AwaitExpr` | `Value Expression` |
| `SpreadExpr` | `Value Expression` |
| `ImportExpr` | `Path string`（编译期特殊形态） |

#### 4.3.2 语句节点

| 节点 | 关键字段 |
|------|---------|
| `VarDeclStmt` | `Names []string`, `Type *TypeAnnotation?`, `Value Expression`, `Kind "let"\|"const"\|"var"` |
| `FunctionDeclStmt` | `Name`, `Params`, `Body`, `IsAsync` |
| `ClassDeclStmt` | `Name`, `SuperClass`, `Body`, `IsAsync` |
| `BlockStmt` | `Statements []Statement` |
| `IfStmt` | `Cond`, `Consequence *BlockStmt`, `Alternative Statement?` |
| `WhileStmt` | `Cond`, `Body *BlockStmt` |
| `ForStmt` | `Init Statement?`, `Cond Expression?`, `Post Expression?`, `Body` |
| `ForInStmt` | `Variable`, `Iterable`, `Body`, `Kind` |
| `ReturnStmt` | `Value Expression?` |
| `BreakStmt` | `Label string?` |
| `ContinueStmt` | `Label string?` |
| `TryStmt` | `Block`, `Catch *CatchClause?`, `Finalizer *BlockStmt?` |
| `ThrowStmt` | `Value Expression` |
| `MatchStmt` | `Expr Expression`, `Arms []*MatchArm`（匹配作为语句） |
| `ExprStmt` | `Expression Expression`（也用于把 `MatchExpr` 当语句用） |
| `ImportDecl` | `Default?`, `Names []ImportSpec`, `Source string` |
| `ExportDecl` | `Decl Statement`（或具名导出） |
| `LabeledStmt` | `Label string`, `Stmt Statement` |

> **没有** `SwitchStmt` / `CaseClause` —— `match` 完全替代它们（见 §4.3.1）。

#### 4.3.3 `match` 表达式与模式

```go
type MatchExpr struct {
    Expr Expression
    Arms []*MatchArm
}

type MatchArm struct {
    Pattern Pattern
    Guard   Expression
    Body    Node  // Expression 或 *BlockStmt
}

type Pattern interface {
    Node
    patternNode()
}

type LiteralPattern    struct { Value Expression }   // 字面量
type IdentifierPattern struct { Name string }         // 绑定
type WildcardPattern   struct {}                      // _
type OrPattern         struct { Alternatives []Pattern }
type RangePattern      struct {
    Start, End Expression
    Inclusive bool   // ..  /  ..=
}
```

匹配求值算法（`Eval(MatchExpr, env)`）：

```
1. subject := Eval(Expr, env)
2. for arm in Arms:
     if matches(arm.Pattern, subject):
        if arm.Guard is nil || truthy(Eval(arm.Guard, env)):
            return Eval(arm.Body, env)
3. throw MatchError("no arm matched")    // 若无 _ 兜底
```

`matches(pattern, value)` 递归实现：
- `LiteralPattern(p)` ⇔ `value === Eval(p, env)`
- `IdentifierPattern(name)` ⇔ `env.Bind(name, value); true`
- `WildcardPattern` ⇔ `true`
- `OrPattern(alts)` ⇔ 存在 `alt` 使 `matches(alt, value)` 为 `true`（注意绑定一致性）
- `RangePattern(s, e, inclusive)` ⇔ value 与两端字面量同类型，且在区间内

#### 4.3.4 类型注解节点

```go
type TypeAnnotation struct {
    Kind     TypeKind
    Name     string                 // "number" "string" ...
    Params   []*TypeAnnotation      // 泛型
    Property map[string]*TypeAnnotation  // 对象类型
    ArrayOf  *TypeAnnotation
    Union    []*TypeAnnotation
    Optional bool
    Return   *TypeAnnotation        // 函数返回类型
}
```

支持的类型：

| 名称 | 含义 |
|------|------|
| `number` | 任意数字（含 NaN、Infinity） |
| `int` | 整数（runtime 检查无小数） |
| `float` | 浮点数 |
| `string` | 字符串 |
| `boolean` | 布尔 |
| `null` `undefined` | 字面量类型 |
| `void` | 仅用于函数返回类型 |
| `any` | 跳过检查 |
| `Array<T>` `T[]` | 数组 |
| `T \| U` | 联合类型 |
| `T?` | 可空（等价 `T \| null`） |
| `{ key: T }` | 对象结构类型 |
| `(a: T) => U` | 函数类型 |

---

## 5. 解析后阶段：解析器（Resolver）

### 5.1 必要性

纯树遍历求值器在每次访问标识符时都要沿着环境链向上查找，**闭包变量捕获**也需要运行时判定。
为了把这一过程前移到静态阶段，并支持类型检查，引入 Resolver：

1. 在 AST 上标注每个 `Identifier` 所属的**作用域深度**与**是否被闭包捕获**。
2. 解析 `this` 在每个函数中的绑定（普通函数、箭头函数、class method 的差异）。
3. 解析 `var` 声明的提升（hoisting）位置。
4. 提前发现对未声明变量的引用、重复声明等语义错误。

### 5.2 作用域表示

```go
type Scope struct {
    parent   *Scope
    vars     map[string]bool
    funcs    map[string]*FunctionDeclStmt
    classes  map[string]*ClassDeclStmt
    thisBinding *Identifier?  // 当前 this
    strict     bool
}
```

### 5.3 闭包捕获

对每个函数体，Resolver 做一次"被引用但定义在外层"的标识符收集，这些变量在编译时被记录为**自由变量**。
求值时，对应的 `*Function` 对象会**按引用捕获外层环境的 `Environment`**，而不是按值拷贝，从而形成真正的闭包。

---

## 6. 类型检查（可选）

> 默认关闭，可通过命令行 `--check-types` 或 API `Evaluator{TypeCheck: true}` 启用。

### 6.1 规则

- **未标注 → 不检查**（保持动态特性）。
- **标注存在 → 运行时求值后比对**。
- 任何不匹配抛 `TypeError`，携带期望类型、实际类型、行列号。

### 6.2 检查点

| 位置 | 检查内容 |
|------|---------|
| `let x: T = expr` | `expr` 的运行时类型是 `T` |
| `function f(p: T)` | 调用时实参类型是 `T` |
| `function f(): T` | 返回值类型是 `T` |
| `class C { f(): T }` | 方法返回值 |
| 赋值 `x: T = expr` | 右侧类型 |
| 字面量上下文 | 仅做断言，不强制 |

> 类型注解本身由 Lexer 扩展支持（`:` 不与对象字面量冲突），Parser 单独提取。

---

## 7. 运行时值与对象系统

### 7.1 `Object` 接口

```go
type Object interface {
    Type() ObjectType
    Inspect() string
    ToGoValue() any        // 暴露给 Go 嵌入
}

type ObjectType string

const (
    NUMBER_OBJ   ObjectType = "NUMBER"
    STRING_OBJ   ObjectType = "STRING"
    BOOLEAN_OBJ  ObjectType = "BOOLEAN"
    NULL_OBJ     ObjectType = "NULL"
    UNDEFINED_OBJ ObjectType = "UNDEFINED"
    ARRAY_OBJ    ObjectType = "ARRAY"
    OBJECT_OBJ   ObjectType = "OBJECT"
    FUNCTION_OBJ ObjectType = "FUNCTION"
    BUILTIN_OBJ  ObjectType = "BUILTIN"
    PROMISE_OBJ  ObjectType = "PROMISE"
    ERROR_OBJ    ObjectType = "ERROR"
    CLASS_OBJ    ObjectType = "CLASS"
    INSTANCE_OBJ ObjectType = "INSTANCE"
    BREAK_OBJ    ObjectType = "BREAK"
    CONTINUE_OBJ ObjectType = "CONTINUE"
    RETURN_OBJ   ObjectType = "RETURN"
)
```

### 7.2 函数对象

```go
type Function struct {
    Name       string
    Parameters []Parameter
    Body       *ast.BlockStmt
    Env        *Environment         // 闭包捕获
    IsAsync    bool
    IsArrow    bool
    TypeSig    *ast.TypeAnnotation   // 可选
}
```

- 普通函数 `this` 绑定到调用者，构造调用 `new f()` 时绑定到新实例。
- 箭头函数 `this` 永久绑定到定义时的外层 `this`（即在 Resolver 阶段闭合）。

### 7.3 类与原型

```
Class C { method() {} }
  └─ ClassObject { name: "C", methods, super }
       └─ Instance { classRef, props: { ...userFields } }
  原型链： instance.__proto__ === C.prototype
        C.prototype.__proto__ === C.super?.prototype
```

- 方法不通过 `this.method = ...` 存储，而由 `*ClassObject` 持有。
- `instance` 访问 `method` 时，沿 `instance.props` → `class.methods` → `super` 链查找。
- 字段（`name: type`）初始化搬到 `constructor`，未显式声明 `constructor` 时插入默认 `constructor(...args) { super(...args) }`。

### 7.4 数组

- 底层为 `[]Object`，支持稀疏数组（`nil` 表示 hole）。
- `length` 始终是 `max(实际长度, 最大索引+1)`。
- 内置方法（`push`/`map`/...）为 `BUILTIN_OBJ`，不写在 `*Array` 上。

### 7.5 字符串

- 不可变 UTF-8。
- 拼接操作 `+` 在 `STRING + STRING` 走快速路径。

### 7.6 错误对象

```go
type Error struct {
    Name    string   // "Error" "TypeError" "RangeError" "ReferenceError" "SyntaxError"
    Message string
    Stack   []StackFrame
}

type StackFrame struct {
    File string
    Line int
    Col  int
    Fn   string  // 函数名
}
```

### 7.7 环境链

```go
type Environment struct {
    store  map[string]Object
    parent *Environment
    fnName string        // 用于堆栈
    line   int
    col    int
}
```

- `var` 提升：词法/解析阶段已记录到最近函数/全局环境。
- `let`/`const`：在 `BlockStmt` 求值时创建新环境。

---

## 8. 求值（Evaluator）

### 8.1 入口

```go
type Evaluator struct {
    globals    *Environment
    resolver   *Resolver
    typeCheck  bool
    asyncRT    *async.Runtime
    // 观测回调
    OnError    func(err *Error)
    OnOutput   func(s string)   // 接管 console
}
```

### 8.2 表达式求值

```go
func (e *Evaluator) Eval(node ast.Node, env *Environment) Object
```

- 模式：`switch n := node.(type) { case *ast.NumberLiteral: ... }`
- 运算符：根据 `Type()` 分发到 `evalNumberInfix`、`evalStringInfix` 等。
- 真值判断：与 JS 相同，`false/0/""/null/undefined/NaN` 为假，其他为真。

### 8.3 控制流

| 节点 | 实现 |
|------|------|
| `BlockStmt` | 创建子环境，顺序执行 |
| `IfStmt` | 求值条件，命中分支 |
| `WhileStmt` | `for { cond; body }`，`break/continue` 通过特殊对象 `BREAK/CONTINUE` 抛出 |
| `ForStmt` | 同上，区分三段 |
| `ForInStmt` | 迭代 `Array` 或 `Object` **自身**的键（不迭代原型链） |
| `MatchStmt` / `MatchExpr` | 见下方 §8.5 |
| `TryStmt` | `defer`-like 栈式保护，进入 catch / finalizer |
| `ThrowStmt` | 包装成 `*Error` 抛出 |
| `ReturnStmt` | 包装成 `*ReturnValue` 抛出 |

### 8.5 `match` 的求值

```go
func (e *Evaluator) evalMatch(node *ast.MatchExpr, env *Environment) Object {
    subject := e.Eval(node.Expr, env)
    if err, ok := subject.(*Error); ok { return err }

    for _, arm := range node.Arms {
        bindings := newScope(env)   // arm 独立的临时作用域
        if matchPattern(arm.Pattern, subject, bindings) {
            if arm.Guard != nil {
                g := e.Eval(arm.Guard, bindings)
                if !isTruthy(g) { continue }
            }
            return e.Eval(arm.Body, bindings)
        }
    }
    return newError("MatchError: no arm matched for %s", subject.Inspect())
}

func matchPattern(p Pattern, v Object, env *Environment) bool {
    switch p := p.(type) {
    case *LiteralPattern:
        return v == e.Eval(p.Value, env)  // 严格相等
    case *IdentifierPattern:
        env.Bind(p.Name, v); return true
    case *WildcardPattern:
        return true
    case *OrPattern:
        for _, alt := range p.Alternatives {
            if matchPattern(alt, v, env) { return true }
        }
        return false
    case *RangePattern:
        s := e.Eval(p.Start, env)
        end := e.Eval(p.End, env)
        if s.Type() != end.Type() || s.Type() != v.Type() { return false }
        if p.Inclusive { return compareLE(s, v) && compareLE(v, end) }
        return compareLE(s, v) && compareLT(v, end)
    }
    return false
}
```

> 绑定一致性：OR 模式中同一名字必须绑定到同一类型（若不同则该 OR arm 不匹配）。

### 8.4 异常传播

求值过程中所有错误（`ReferenceError`、`TypeError`、用户 `throw`）都通过 `panic(*Error)` 传播。
外层 `try-catch` 在求值 `Block` 入口处包 `defer recover()` 捕获。

### 8.5 性能取舍

- **同步热路径**：`NUMBER <op> NUMBER`、`STRING + STRING` 等不分配中间 `[]Object`，直接返回新值。
- **避免反射**：`ToGoValue` 仅在 `BUILTIN_OBJ` 把控制权交还 Go 时使用。
- **短对象池**：`Error`、`ReturnValue` 等小对象使用 `sync.Pool` 复用。

---

## 9. 异步模型

### 9.1 关键事实

> **挑战**：树遍历求值器在执行 `await` 时需要"暂停并稍后继续"。
> 解决：把"await 之后的语句序列"封装为 **continuation thunk**，注册到 `Promise.then()`。

### 9.2 事件循环

```go
type Runtime struct {
    microtask  queue.Queue[Task]   // 微任务队列
    macrotask  queue.Queue[Task]   // 宏任务队列
    pending    map[int]*Promise
    nextID     int64
    isRunning  bool
}

type Task struct {
    fn     func() Object
    env    *Environment
}
```

#### 9.2.1 循环步骤

```
Tick():
  1. 执行所有微任务直到清空
  2. 执行一个宏任务
  3. 回到 1
  4. 当两队列都空且无 pending 异步 I/O，结束运行
```

### 9.3 Promise

```go
type Promise struct {
    state     State          // PENDING | FULFILLED | REJECTED
    value     Object
    handlers  []Handler      // 链式回调
    microtasks []Task
    id        int64
}
```

- `new Promise(executor)`：同步执行 `executor`，捕获内部 `resolve` / `reject`。
- `then(onFulfilled, onRejected?)`：
  - 若 `state == PENDING` → 注册到 `handlers`。
  - 若已决 → 直接入微任务。
- 决议：把 `value` / `reason` 写入，遍历 `handlers` 转为微任务。

### 9.4 async/await 编译形态

在 Resolver 阶段为 `async` 函数生成**带状态机的求值骨架**：

```text
async function f() { S1; await p; S2; await q; S3; }
≈ 等价于
function f() {
  return new Promise((resolve, reject) => {
    state = 0;
    cont = (v) => {
      try {
        switch (state) {
          case 0: _v = eval(S1); state = 1; return Promise.resolve(_v).then(p).then(cont, reject);
          case 1: _v = p_value;        state = 2; eval(S2); state = 3; ...
        }
      } catch (e) { reject(e); }
    };
    cont(undefined);
  });
}
```

**实现策略**（v0.1 简化）：

- 不做静态 CPS 变换，改为在 `Eval(AwaitExpr, env)` 中：
  1. 先求值 `await.Value`。
  2. 若结果是 `*Promise` 且 `state == PENDING`：
     - 把 `AwaitExpr` 之后的"续延"作为一个 closure 保存在 `Promise.handlers`。
     - 返回一个"挂起占位"的 `*Promise` 给上层。
  3. 若 `state` 已决，直接取值继续。

> 该策略等效于把 `await` 实现为 `Promise.then(cont)`，其中 `cont` 是一段 closure：它持有 `env` 与"接下来要执行的语句序列"。

### 9.5 宿主集成

- `setTimeout(fn, ms)`：由 Go 端 `time.AfterFunc` 触发，把 `fn` 包成宏任务入队。
- `setInterval(fn, ms)`：循环注册 `time.AfterFunc`。
- 网络 I/O（`fetch`）：使用 `net/http`，回调进入宏任务。
- 控制流：`Evaluator.Run()` 在同步代码结束后**进入 `Runtime.RunLoop()`**，直到所有任务结束。

### 9.6 错误传播

- `async` 函数中的 `throw` 自动转 `reject`。
- `try/catch` 可跨越 `await` 捕获。
- 未捕获的 `reject` 在宏任务结束时通过 `OnError` 上报。

---

## 10. 标准库

详见 [`docs/builtins.md`](builtins.md)。

| 命名空间 | 提供 |
|---------|------|
| `console` | `log`, `error`, `warn`, `info`, `debug`, `time`, `timeEnd` |
| `Math` | `abs`, `floor`, `ceil`, `round`, `trunc`, `random`, `max`, `min`, `pow`, `sqrt`, `sin`, `cos`, `tan`, `log`, `log2`, `log10`, `PI`, `E` |
| `JSON` | `stringify`, `parse` |
| `Object` | `keys`, `values`, `entries`, `assign`, `create`, `freeze`, `isFrozen` |
| `Array` | `isArray`（其余由 `*Array` 实例方法提供） |
| `String` / `Number` / `Boolean` | 构造包装；`String` / `Number` 提供静态方法 `parseInt` / `parseFloat` 等 |
| `Promise` | `resolve`, `reject`, `all`, `race`, `allSettled` |
| `setTimeout` / `setInterval` | 顶层函数 |

---

## 11. 错误报告

### 11.1 错误模型

```go
type RuntimeError struct {
    Type     string     // "TypeError" "ReferenceError" "RangeError" "SyntaxError" "Error"
    Message  string
    Stack    []Frame
    Pos      Position
}

type Frame struct {
    Fn   string
    File string
    Line int
    Col  int
}
```

### 11.2 输出格式

```
TypeError: cannot read property 'name' of undefined
    at User.greet  (./user.gs:12:9)
    at main        (./app.gs:5:1)
    at <script>    (./app.gs:1:1)
```

### 11.3 编译期错误

- 词法错误：`SyntaxError: unexpected character '@' at line 3 col 5`
- 语法错误：`SyntaxError: expected '}' after block, got 'EOF' at line 10 col 1`
- 解析错误：`ReferenceError: variable 'foo' is not declared at line 1 col 4`

---

## 12. 嵌入 API（Go 侧）

```go
import "github.com/yourname/goscript/internal/evaluator"
import "github.com/yourname/goscript/internal/parser"
import "github.com/yourname/goscript/internal/lexer"

func RunScript(source string) error {
    l := lexer.New("<embed>", source)
    p := parser.New(l)
    program := p.ParseProgram()
    if len(p.Errors()) > 0 { return p.Errors() }

    e := evaluator.New()
    e.TypeCheck = true
    e.OnOutput = func(s string) { fmt.Print(s) }

    result := e.Eval(program, e.Globals())
    if err, ok := result.(*object.Error); ok {
        return err
    }
    return nil
}
```

未来提供 `evaluator.New().EvalString(src)` 一键入口。

---

## 13. 关键算法与权衡

| 决策 | 选项 | 选择 | 理由 |
|------|------|------|------|
| 解释执行 | AST 树遍历 vs 字节码 VM | **AST 树遍历** | 用户指定；代码清晰；教学价值高；性能对脚本场景足够 |
| 表达式解析 | 递归下降 vs Pratt | **Pratt** | 优先级表驱动，新增运算符成本低 |
| 数字模型 | int / float / 通用 | **统一 float64** | 与 JS 一致；`int` 通过类型注解标记 |
| 闭包实现 | 平展 vs 环境指针 | **环境指针** | 真正共享可变状态，符合预期 |
| 异步实现 | goroutine + channel vs continuation | **continuation** | 树遍历友好；调试可控 |
| 类型注解 | 编译期擦除 vs 运行时检查 | **运行时检查** | 树遍历解释器统一阶段；与"可选"语义匹配 |
| 错误恢复 | 遇错即停 vs 多错聚合 | **多错聚合** | 体验更好 |
| 模块系统 | ES Module 静态分析 vs 动态 require | **动态 require + import 语法糖** | 简单可实现 |
| 输出 | fmt 直打 vs 回调 | **回调** | 嵌入友好 |

---

## 14. 测试策略

| 层 | 方法 |
|----|------|
| 词法 | 输入字符串 → 期望 token 序列（表驱动） |
| 语法 | 输入字符串 → AST dump 快照（golden file） |
| 解析 | 自由变量收集正确性 |
| 类型 | 注解正确/错误用例 |
| 求值 | `.gs` 脚本跑出期望输出 |
| 异步 | `setTimeout` 顺序、`Promise` 链、`await` 挂起恢复 |
| 错误 | 异常类型、行号、堆栈格式 |
| 嵌入 | Go 测试驱动脚本 |

---

## 15. 实施顺序（建议）

1. **M0 骨架**：`cmd/gs/main.go`、项目目录、`go.mod`
2. **M1 词法**：所有 token，覆盖数字 / 字符串 / 模板
3. **M2 语法**：表达式 + 基础语句（let / if / while / return）
4. **M3 求值器**：基础类型、运算符、控制流
5. **M4 函数 / 闭包**：一等公民、闭包捕获
6. **M5 数组 / 对象 / 内置方法**
7. **M6 class / extends / super / this 绑定**
8. **M7 try / throw / Error 对象**
9. **M8 解析器**：作用域、闭包变量标注、this 静态绑定
10. **M9 异步**：Promise、async/await、事件循环
11. **M10 类型注解**
12. **M11 模块系统**
13. **M12 标准库补全 + REPL**
14. **M13 嵌入 API 与文档打磨**

---

## 16. 风险与开放问题

| 风险 | 缓解 |
|------|------|
| `await` 在深层控制流（`if/while`）中的续延构建复杂度 | 维护一份"续延 builder"统一工具 |
| 性能不足 | 后续可加入"热点函数字节码编译"作为可选加速层 |
| `class` 私有字段（`#x`）的语法复杂度 | v0.1 不支持，文档注明 |
| Unicode 标识符 | 区分"标识符字符"使用 `unicode.IsLetter`/`IsDigit` |
| 浮点相等性 | 接受 `1/3*3 !== 1` 的 JS 行为（`NaN === NaN` 仍为 `true`，与 IEEE-754 偏离） |
| `with` / `eval` / `arguments` | 不支持（v0.1，避免实现复杂度） |
| `switch` 的"用户从 JS 迁移" | 已有 [`docs/bad-parts-fixed.md`](bad-parts-fixed.md) §10 提供 `switch→match` 改写示例 |
| Generators / `for await` | v0.2 评估 |
| Match 模式中 OR 分支的绑定一致性 | 编译期检查同一 OR 内同名绑定必须出现在所有分支 |

---

> 文档结束。下一步：阅读 [`docs/language-spec.md`](language-spec.md) 了解完整语言特性，
> [`docs/bad-parts-fixed.md`](bad-parts-fixed.md) 了解被修复的 JS 缺点，
> 或 [`docs/grammar.ebnf`](grammar.ebnf) 查看形式化语法。
