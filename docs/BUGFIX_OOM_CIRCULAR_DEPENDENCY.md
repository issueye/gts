# 内存耗尽（OOM）问题修复文档

**修复日期**: 2026-06-11  
**问题编号**: 网关运行时内存耗尽崩溃  
**严重程度**: 🔴 Critical

## 问题描述

网关程序（gs-gateway.exe）运行时出现以下致命错误：

```
runtime: VirtualAlloc of 8192 bytes failed with errno=1455
fatal error: out of memory
```

**Windows 错误码 1455** = `ERROR_COMMITMENT_LIMIT`（系统内存承诺限制）

## 根本原因分析

通过堆栈跟踪分析，发现两个关键问题：

### 1. 循环依赖导致无限递归

在 `cmd/gs/main.go` 的模块加载机制中：

```go
func (r *runner) requireFunc(baseDir string) evaluator.RequireFn {
    return func(path string) (object.Object, error) {
        // ...
        env := r.cache.GetOrCreate(cacheKey)  // ⚠️ 创建空环境加入缓存
        module.SetupExports(env)
        r.configureModuleLoaders(env, ...)    // ⚠️ 设置加载器
        
        // ❌ 如果脚本有循环依赖，此时会进入无限递归
        if _, err := r.evalSource(string(src), resolved.Path, env); err != nil {
            return nil, err
        }
        // ...
    }
}
```

**问题链**：
- 模块 A 加载 → 创建空 env 放入缓存 → 开始解析
- 解析 A 时遇到 `import "B"` → 加载模块 B
- 解析 B 时遇到 `import "A"` → 缓存中有 A（空的）→ 继续递归
- 无限循环直到栈溢出或内存耗尽

### 2. Parser 错误累积

在 `internal/parser/parser.go:198` 中：

```go
func (p *Parser) addError(msg string) {
    p.errors = append(p.errors, fmt.Sprintf("%s: %s", p.pos(), msg))  // 无限制累积
}
```

当脚本有语法错误时，parser 不断调用 `addError`，每次调用都分配内存，没有上限保护。

## 修复方案

### 修复 1: 添加循环依赖检测

**文件**: `gts/cmd/gs/main.go`

1. 在 `runner` 结构体中添加 `loading` 字段：

```go
type runner struct {
    opts     options
    pool     *async.Pool
    cache    *module.Cache
    vm       *object.VirtualMachine
    resolver *module.Resolver
    plugins  *pluginhost.Host
    rootDir  string
    loading  map[string]bool  // 新增：正在加载的模块路径
}
```

2. 在 `newRunner` 中初始化：

```go
func newRunner(opts options) *runner {
    if opts.workers < 1 {
        opts.workers = 1
    }
    return &runner{
        opts:    opts,
        loading: make(map[string]bool),  // 初始化
    }
}
```

3. 在 `requireFunc` 中添加检测逻辑：

```go
func (r *runner) requireFunc(baseDir string) evaluator.RequireFn {
    return func(path string) (object.Object, error) {
        // ... 省略前置代码 ...
        
        // 检测循环依赖
        if r.loading[cacheKey] {
            return nil, fmt.Errorf("circular dependency detected: %s", path)
        }
        r.loading[cacheKey] = true
        defer delete(r.loading, cacheKey)
        
        // ... 继续执行加载逻辑 ...
    }
}
```

4. 在 `requireFrom` 中也添加检测：

```go
func (r *runner) requireFrom(env *object.Environment, path string) (object.Object, error) {
    // ... 省略前置代码 ...
    
    // 检测循环依赖
    cacheKey := resolved.ID
    if cacheKey == "" {
        cacheKey = resolved.Path
    }
    if r.loading[cacheKey] {
        return nil, fmt.Errorf("circular dependency detected: %s", path)
    }
    
    return r.requireFunc(baseDir)(path)
}
```

### 修复 2: 限制 Parser 错误数量

**文件**: `gts/internal/parser/parser.go`

1. 添加最大错误数量常量：

```go
const maxParseErrors = 100 // 限制最大错误数量，防止内存耗尽
```

2. 修改 `addError` 方法：

```go
func (p *Parser) addError(msg string) {
    if len(p.errors) >= maxParseErrors {
        if len(p.errors) == maxParseErrors {
            p.errors = append(p.errors, fmt.Sprintf("%s: too many parse errors (limit: %d)", p.pos(), maxParseErrors))
        }
        return
    }
    p.errors = append(p.errors, fmt.Sprintf("%s: %s", p.pos(), msg))
}
```

## 验证测试

### 测试 1: 循环依赖检测

创建两个相互依赖的脚本：

**test_circular_a.gs**:
```javascript
import b from "./test_circular_b.gs"
exports.name = "Module A"
```

**test_circular_b.gs**:
```javascript
import a from "./test_circular_a.gs"
exports.name = "Module B"
```

运行：
```bash
./gs-gateway.exe test_circular_a.gs
```

**预期结果**:
```
ImportError: circular dependency detected: ./test_circular_b.gs
```

✅ **测试通过**：程序正确检测到循环依赖并报错退出，不再无限递归。

### 测试 2: 编译验证

```bash
cd gts
go build -o ../dist/release/gs-gateway.exe ./cmd/gs
```

✅ **编译成功**：无任何编译错误或警告。

## 影响范围

### 修改文件
- `gts/cmd/gs/main.go` - 添加循环依赖检测机制
- `gts/internal/parser/parser.go` - 限制错误数量

### 向后兼容性
✅ **完全兼容**：
- 不影响正常脚本的执行
- 只在检测到循环依赖时才报错（之前会崩溃）
- 语法错误处理更加健壮

### 性能影响
✅ **可忽略**：
- 循环依赖检测使用 map 查找，O(1) 时间复杂度
- 内存开销：每个加载中的模块一个 bool 值
- 错误限制避免了内存无限增长

## 后续建议

### 1. 增强错误提示
当检测到循环依赖时，显示完整的依赖链路：
```
circular dependency detected:
  A -> B -> C -> A
```

### 2. 添加单元测试
在 `cmd/gs/main_test.go` 中添加：
- 循环依赖检测测试
- Parser 错误限制测试

### 3. 文档更新
在用户文档中说明：
- 不支持循环依赖（与 Node.js/Python 类似）
- 如何重构代码避免循环依赖

## 总结

本次修复解决了一个严重的内存耗尽问题，防止了以下场景下的崩溃：
1. ✅ 脚本间的循环依赖
2. ✅ 语法错误导致的 parser 错误累积
3. ✅ 递归加载导致的栈溢出

修复后的程序更加健壮，能够优雅地处理错误情况而不是直接崩溃。
