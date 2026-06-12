# GTS 原生标准库开发计划

> 本文档详细规划标准库扩展的实施步骤、时间安排和技术细节。
> 创建时间：2026-06-12

---

## 一、开发策略

### 1.1 总体原则

- **模块优先**：优先完成 P0 级别的 4 个模块
- **文档先行**：先写知识库文档，再实现代码
- **测试驱动**：每个模块都有完整的测试用例
- **示例同步**：每个模块都提供实际使用示例
- **渐进式交付**：每完成一个模块立即可用

### 1.2 开发流程

每个模块的开发遵循以下步骤：

```
1. 创建知识库文档 (docs/wiki/native-stdlib/xxx.md)
2. 实现 Go 侧模块 (internal/stdlib/mod_xxx.go)
3. 注册模块 (internal/stdlib/stdlib.go)
4. 编写单元测试 (internal/stdlib/mod_xxx_test.go)
5. 创建示例脚本 (examples/xxx.gs)
6. 更新主文档索引 (docs/wiki/native-stdlib.md)
```

### 1.3 技术栈

- **Go 实现**：所有原生模块用 Go 实现，提供最佳性能
- **对象封装**：使用 `object.Object` 接口封装返回值
- **错误处理**：统一的错误处理机制
- **异步支持**：需要时支持 Promise 返回

---

## 二、第一阶段：P0 模块（1-2 周）

### 2.1 模块 1：`@std/test` - 测试框架

**优先级**：🔴 最高  
**预计工作量**：3-4 天  
**负责人**：待分配

#### 实现细节

**文件结构**：
```
internal/stdlib/
  mod_test.go           # 主模块实现
  mod_test_expect.go    # 断言实现
  mod_test_runner.go    # 测试运行器
  mod_test_test.go      # 单元测试

docs/wiki/native-stdlib/
  test.md               # 知识库文档

examples/
  25-test-framework.gs  # 示例脚本
```

**核心数据结构**：
```go
type TestSuite struct {
    Name      string
    Tests     []*TestCase
    Hooks     *TestHooks
    Parent    *TestSuite
    Children  []*TestSuite
}

type TestCase struct {
    Name     string
    Fn       func() error
    Async    bool
    Skip     bool
    Only     bool
    Timeout  time.Duration
}

type TestHooks struct {
    BeforeAll   []func() error
    AfterAll    []func() error
    BeforeEach  []func() error
    AfterEach   []func() error
}

type TestResult struct {
    Total   int
    Passed  int
    Failed  int
    Skipped int
    Errors  []TestError
}
```

**模块导出**：
```go
func createTestModule() *object.Instance {
    return &object.Instance{
        Props: map[string]object.Object{
            "describe":    // 创建测试套件
            "it":          // 创建测试用例
            "test":        // it 的别名
            "beforeAll":   // 钩子
            "afterAll":
            "beforeEach":
            "afterEach":
            "expect":      // 断言
            "skip":
            "only":
            "run":         // 运行测试
            "configure":   // 配置
        },
    }
}
```

**断言实现**：
```go
type Expectation struct {
    Value    object.Object
    Not      bool
}

func (e *Expectation) ToBe(expected object.Object) error
func (e *Expectation) ToEqual(expected object.Object) error
func (e *Expectation) ToBeTruthy() error
// ... 更多断言方法
```

**测试任务**：
- ✅ 基础测试套件创建
- ✅ 同步测试执行
- ✅ 异步测试执行
- ✅ 断言库
- ✅ 钩子函数
- ✅ skip/only 功能
- ✅ 测试报告生成

**示例**：
```javascript
// examples/25-test-framework.gs
let test = require("@std/test");

test.describe("Math", () => {
  test.it("should add", () => {
    test.expect(1 + 1).toBe(2);
  });
  
  test.it("should handle async", async () => {
    let result = await Promise.resolve(42);
    test.expect(result).toBe(42);
  });
});

test.run();
```

---

### 2.2 模块 2：`@std/env` - 环境变量管理

**优先级**：🔴 高  
**预计工作量**：2 天  
**负责人**：待分配

#### 实现细节

**文件结构**：
```
internal/stdlib/
  mod_env.go
  mod_env_parser.go     # .env 文件解析
  mod_env_test.go

docs/wiki/native-stdlib/
  env.md

examples/
  26-env-management.gs
```

**核心功能**：
```go
// .env 文件解析器
type EnvParser struct{}

func (p *EnvParser) Parse(content string) (map[string]string, error)
func (p *EnvParser) ParseFile(path string) (map[string]string, error)

// 类型转换
func getInt(key string, defaultVal int) int
func getBool(key string, defaultVal bool) bool
func getArray(key string, sep string) []string
```

**模块导出**：
```go
func createEnvModule() *object.Instance {
    return &object.Instance{
        Props: map[string]object.Object{
            "load":         // 加载 .env
            "loadMultiple": // 加载多个文件
            "get":          // 获取字符串
            "getString":
            "getInt":
            "getFloat":
            "getBool":
            "getArray":
            "getJson":
            "require":      // 验证必需变量
            "has":
            "set":
            "unset":
            "toObject":
            "parse":        // 解析内容
        },
    }
}
```

**测试任务**：
- ✅ .env 文件解析
- ✅ 多行值支持
- ✅ 注释处理
- ✅ 引号处理
- ✅ 变量展开
- ✅ 类型转换
- ✅ 多文件加载
- ✅ 验证功能

---

### 2.3 模块 3：`@std/json` - JSON 增强

**优先级**：🔴 高  
**预计工作量**：3 天  
**负责人**：待分配

#### 实现细节

**文件结构**：
```
internal/stdlib/
  mod_json.go
  mod_json5.go          # JSON5 解析
  mod_json_schema.go    # Schema 验证
  mod_json_pointer.go   # JSON Pointer
  mod_json_patch.go     # JSON Patch
  mod_json_test.go

docs/wiki/native-stdlib/
  json.md

examples/
  27-json-enhanced.gs
```

**依赖库**（Go）：
- `github.com/flynn/json5` - JSON5 解析
- `github.com/xeipuuv/gojsonschema` - JSON Schema
- `github.com/evanphx/json-patch` - JSON Patch

**核心功能**：
```go
// JSON5
func parse5(text string) (object.Object, error)
func stringify5(obj object.Object, opts map[string]interface{}) (string, error)

// Schema 验证
type SchemaValidator struct {
    schema *gojsonschema.Schema
}
func (v *SchemaValidator) Validate(data interface{}) ValidationResult

// JSON Pointer
func jsonGet(doc interface{}, pointer string) (interface{}, error)
func jsonSet(doc interface{}, pointer string, value interface{}) error

// JSON Patch
func jsonPatch(doc interface{}, patch []PatchOp) (interface{}, error)
func jsonDiff(oldDoc, newDoc interface{}) []PatchOp
```

**测试任务**：
- ✅ JSON5 解析（注释、尾逗号、单引号）
- ✅ JSON5 序列化
- ✅ Schema 验证
- ✅ JSON Pointer 操作
- ✅ JSON Patch 应用
- ✅ JSON Diff 生成

---

### 2.4 模块 4：`@std/validation` - 数据验证

**优先级**：🔴 高  
**预计工作量**：3-4 天  
**负责人**：待分配

#### 实现细节

**文件结构**：
```
internal/stdlib/
  mod_validation.go
  mod_validation_types.go      # 类型定义
  mod_validation_validators.go # 验证器实现
  mod_validation_test.go

docs/wiki/native-stdlib/
  validation.md

examples/
  28-validation.gs
```

**核心数据结构**：
```go
type Validator interface {
    Validate(value object.Object) ValidationResult
}

type StringValidator struct {
    minLength    *int
    maxLength    *int
    pattern      *regexp.Regexp
    email        bool
    url          bool
    uuid         bool
    required     bool
    defaultValue *string
}

type NumberValidator struct {
    min       *float64
    max       *float64
    integer   bool
    positive  bool
    negative  bool
    multiple  *float64
    required  bool
}

type ArrayValidator struct {
    elementType Validator
    minLength   *int
    maxLength   *int
    unique      bool
    required    bool
}

type ObjectValidator struct {
    shape    map[string]Validator
    required bool
}

type ValidationResult struct {
    Valid  bool
    Value  object.Object
    Errors []ValidationError
}

type ValidationError struct {
    Path    string
    Message string
    Value   object.Object
}
```

**模块导出**：
```go
func createValidationModule() *object.Instance {
    return &object.Instance{
        Props: map[string]object.Object{
            "string":  createStringValidator,
            "number":  createNumberValidator,
            "boolean": createBooleanValidator,
            "array":   createArrayValidator,
            "object":  createObjectValidator,
            "date":    createDateValidator,
            "any":     createAnyValidator,
        },
    }
}
```

**链式 API 实现**：
```go
func (v *StringValidator) Min(n int) *StringValidator {
    v.minLength = &n
    return v
}

func (v *StringValidator) Max(n int) *StringValidator {
    v.maxLength = &n
    return v
}

func (v *StringValidator) Email() *StringValidator {
    v.email = true
    return v
}

// 返回 JS 对象时包装为可调用
func wrapStringValidator(v *StringValidator) *object.Instance {
    return &object.Instance{
        Props: map[string]object.Object{
            "min":      &object.Builtin{Fn: ...},
            "max":      &object.Builtin{Fn: ...},
            "email":    &object.Builtin{Fn: ...},
            "validate": &object.Builtin{Fn: ...},
            "parse":    &object.Builtin{Fn: ...},
        },
    }
}
```

**测试任务**：
- ✅ 字符串验证（长度、模式、格式）
- ✅ 数字验证（范围、整数、正负）
- ✅ 数组验证（元素类型、长度、唯一性）
- ✅ 对象验证（形状、嵌套）
- ✅ 链式 API
- ✅ 可选与默认值
- ✅ 自定义验证
- ✅ 错误信息

---

## 三、第二阶段：P1 模块（2-3 周）

### 3.1 模块 5：`@std/collections` - 集合操作

**预计工作量**：2 天

**核心实现**：
```go
func groupBy(arr []object.Object, fn func(object.Object) string) map[string][]object.Object
func unique(arr []object.Object) []object.Object
func chunk(arr []object.Object, size int) [][]object.Object
func difference(a, b []object.Object) []object.Object
func intersection(a, b []object.Object) []object.Object
```

---

### 3.2 模块 6：`@std/random` - 随机数增强

**预计工作量**：1-2 天

**依赖**：`crypto/rand` 提供密码学安全随机

**核心实现**：
```go
func randomInt(min, max int) int
func randomBytes(n int) []byte
func randomHex(n int) string
func randomUUID() string
func shuffle(arr []object.Object) []object.Object
```

---

### 3.3 模块 7：`@std/color` - 终端颜色

**预计工作量**：1-2 天

**依赖**：
- `github.com/fatih/color` 或自行实现 ANSI 码

**核心实现**：
```go
type Color struct {
    codes []int
}

func (c *Color) Sprint(s string) string {
    return fmt.Sprintf("\x1b[%dm%s\x1b[0m", c.codes, s)
}

func createColorFunc(code int) func(string) string
```

---

### 3.4 模块 8：`@std/semver` - 版本管理

**预计工作量**：2 天

**依赖**：
- `github.com/Masterminds/semver/v3`

**核心实现**：
```go
func parse(versionStr string) (*semver.Version, error)
func compare(v1, v2 string) int
func satisfies(version, constraint string) bool
func inc(version, release string) string
```

---

### 3.5 模块 9：`@std/cache` - 内存缓存

**预计工作量**：2 天

**核心实现**：
```go
type Cache struct {
    items   map[string]*CacheItem
    mu      sync.RWMutex
    max     int
    ttl     time.Duration
    lru     *list.List
    stats   *CacheStats
}

type CacheItem struct {
    key       string
    value     object.Object
    expireAt  time.Time
    element   *list.Element
}

func (c *Cache) Set(key string, value object.Object, ttl time.Duration)
func (c *Cache) Get(key string) (object.Object, bool)
```

---

## 四、第三阶段：P2 模块（按需实施）

P2 模块根据用户反馈和实际需求按优先级实施，每个模块 1-3 天工作量。

建议顺序：
1. `@std/diff` - 文本差异（依赖 `github.com/sergi/go-diff`）
2. `@std/glob` - 高级匹配（依赖 `github.com/gobwas/glob`）
3. `@std/watch` - 文件监听（依赖 `github.com/fsnotify/fsnotify`）
4. `@std/retry` - 重试逻辑（纯 Go 实现）
5. `@std/rate-limit` - 速率限制（纯 Go 实现）
6. 其余模块根据需求排期

---

## 五、实施时间表

### 第 1 周：P0 模块（1-4）

| 天数 | 任务 | 产出 |
|------|------|------|
| 周一 | `@std/test` 设计与文档 | 知识库文档 |
| 周二 | `@std/test` 实现与测试 | 代码 + 测试 |
| 周三 | `@std/env` 完整实现 | 代码 + 测试 + 示例 |
| 周四 | `@std/json` 设计与实现（一半） | 基础功能 |
| 周五 | `@std/json` 完成与测试 | 完整模块 |

### 第 2 周：P0 模块（4）+ P1 模块（5-7）

| 天数 | 任务 | 产出 |
|------|------|------|
| 周一 | `@std/validation` 设计 | 知识库文档 |
| 周二 | `@std/validation` 实现（一半） | 基础验证器 |
| 周三 | `@std/validation` 完成 | 完整模块 |
| 周四 | `@std/collections` 实现 | 完整模块 |
| 周五 | `@std/random` 实现 | 完整模块 |

### 第 3 周：P1 模块（7-9）

| 天数 | 任务 | 产出 |
|------|------|------|
| 周一 | `@std/color` 实现 | 完整模块 |
| 周二-周三 | `@std/semver` 实现 | 完整模块 |
| 周四-周五 | `@std/cache` 实现 | 完整模块 |

### 第 4 周及之后：P2 模块

根据实际需求和优先级排期。

---

## 六、质量保证

### 6.1 代码规范

- 遵循 Go 标准代码风格
- 所有导出函数都有注释
- 复杂逻辑添加内联注释
- 使用有意义的变量名

### 6.2 测试要求

每个模块必须包含：
- ✅ 单元测试（覆盖率 > 80%）
- ✅ 集成测试（与 evaluator 集成）
- ✅ 示例脚本（examples/*.gs）
- ✅ 错误处理测试
- ✅ 边界条件测试

### 6.3 文档要求

每个模块必须提供：
- ✅ 知识库文档（docs/wiki/native-stdlib/*.md）
- ✅ API 参考（每个函数的签名和说明）
- ✅ 使用示例（至少 3-5 个）
- ✅ 常见用例（结合实际场景）
- ✅ 错误处理说明

### 6.4 性能要求

- 避免不必要的内存分配
- 合理使用并发（如测试框架）
- 提供基准测试（*_test.go 中的 Benchmark）
- 大数据处理考虑流式 API

---

## 七、依赖管理

### 7.1 新增 Go 依赖

```bash
# JSON5
go get github.com/flynn/json5

# JSON Schema
go get github.com/xeipuuv/gojsonschema

# JSON Patch
go get github.com/evanphx/json-patch/v5

# Semver
go get github.com/Masterminds/semver/v3

# Color（可选，可自行实现）
go get github.com/fatih/color

# Diff
go get github.com/sergi/go-diff/diffmatchpatch

# Glob
go get github.com/gobwas/glob

# Watch
go get github.com/fsnotify/fsnotify
```

### 7.2 依赖原则

- 优先使用 Go 标准库
- 选择维护活跃、Stars 多的第三方库
- 避免过度依赖，保持项目轻量
- 记录每个依赖的用途和版本

---

## 八、风险与应对

### 8.1 技术风险

| 风险 | 影响 | 应对 |
|------|------|------|
| 第三方库不稳定 | 模块功能异常 | 选择成熟库，添加兜底逻辑 |
| 性能瓶颈 | 大数据场景慢 | 提供流式 API，添加性能测试 |
| API 设计不当 | 用户体验差 | 参考成熟生态，早期征求反馈 |
| 测试覆盖不足 | 隐藏 Bug | 强制测试覆盖率，增加边界测试 |

### 8.2 时间风险

| 风险 | 应对 |
|------|------|
| 工作量估算偏差 | 留出 20% 缓冲时间 |
| 并发开发冲突 | 明确模块边界，减少耦合 |
| 需求变更 | P0 锁定需求，P1/P2 灵活调整 |

---

## 九、验收标准

每个模块完成时必须满足：

### 9.1 功能完整性
- ✅ 所有设计的 API 都已实现
- ✅ 核心用例都有示例覆盖
- ✅ 错误处理完善

### 9.2 质量达标
- ✅ 单元测试通过（覆盖率 > 80%）
- ✅ 示例脚本可运行
- ✅ 无 Go vet 警告
- ✅ 代码审查通过

### 9.3 文档齐全
- ✅ 知识库文档完整
- ✅ API 说明清晰
- ✅ 示例代码可复制运行
- ✅ 已更新主索引文档

### 9.4 集成验证
- ✅ 可通过 `require("@std/xxx")` 加载
- ✅ 与现有模块无冲突
- ✅ examples/ 中有对应示例
- ✅ 主 README 已更新

---

## 十、后续迭代

### 10.1 用户反馈收集

- 通过 GitHub Issues 收集功能请求
- 关注高频使用模块的性能问题
- 收集 API 易用性反馈

### 10.2 持续优化

- 每月回顾一次模块使用情况
- 根据反馈调整 API 设计
- 优化性能瓶颈
- 补充缺失的边界用例

### 10.3 版本规划

- v0.2：P0 + P1 模块完成
- v0.3：P2 模块按需完成
- v1.0：所有规划模块稳定

---

## 十一、开始实施

✅ 规范设计已完成  
✅ 开发计划已制定  
🚀 **下一步：开始实施第一阶段 P0 模块**

执行命令：
```bash
# 创建第一个模块的知识库文档
# 实施 @std/test 测试框架
```

---

**文档维护**：本计划随实施进度持续更新，记录实际进度和调整。
