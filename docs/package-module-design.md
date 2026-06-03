# 包模块打包与引用设计

> 目标：让 GoScript 项目可以组织为可复用包，支持稳定的包内引用、包间依赖、离线打包和单文件分发，同时保持当前 `require` / `import` / native module 行为可迁移。

## 1. 当前现状

- 运行时加载由 CLI 注入的 `require(path)` 和 `env.VM().Import(...)` 完成。
- `internal/module.Cache` 以解析后的绝对文件路径缓存模块环境，同一 VM 内避免重复执行。
- `module.ResolvePath(path, baseDir)` 当前支持：
  - 绝对路径
  - 相对路径
  - `@agent/*` 到项目根 `scripts/agent/*`
  - native module 由 `module.GetNative("@std/...")` 提前短路
- `internal/bundle` 目前是实验性源码拼接器，只扫描静态 `require("...")`，尚未与运行时 resolver 共用规则。
- `project.toml` 当前只读取 `[project] name/version/entry`。

这些能力已经能跑多文件项目，但还缺少包级元数据、依赖锁定、包名引用、包产物格式和统一 resolver。

## 2. 设计目标

1. **统一解析**：运行时、打包器、未来类型检查器使用同一套 module specifier 解析规则。
2. **包级组织**：一个目录可以声明为包，拥有入口、导出映射、依赖和构建配置。
3. **离线可复现**：依赖解析写入 lock 文件，打包时不依赖网络或隐式状态。
4. **VM 隔离一致**：包模块加载后仍在当前 VM 中运行；不同 VM 的 cache、对象和 hook 不共享。
5. **兼容当前代码**：保留相对路径、`@std/*`、`@agent/*`、CommonJS 风格 `require` 和 ES `import/export`。
6. **打包可分发**：支持把项目打成单文件脚本或包归档，供 CLI 或嵌入式运行时加载。

## 3. 术语

| 术语 | 含义 |
|------|------|
| `specifier` | 源码中写的引用字符串，如 `"./lib"`、`"@std/fs"`、`"@agent/core"`、`"pkg:foo"` |
| `package root` | 包清单所在目录 |
| `manifest` | 包清单，建议仍使用 `project.toml`，新增 `[package]`、`[exports]`、`[dependencies]` |
| `module id` | resolver 输出的稳定模块身份，作为 cache key |
| `source module` | 普通 `.gs` 文件模块 |
| `native module` | Go 侧注册的 `@std/*` 模块 |
| `package module` | 由包名和导出映射解析出来的模块 |
| `bundle` | 解析依赖图后的可执行单文件或归档产物 |

## 4. 包清单格式

沿用 `project.toml`，避免新增多个顶层文件。推荐格式：

```toml
[project]
name = "hello-agent"
version = "0.1.0"
entry = "src/main.gs"

[package]
name = "@local/hello-agent"
version = "0.1.0"
type = "script"
main = "src/main.gs"

[exports]
"." = "src/index.gs"
"./tools" = "src/tools/index.gs"
"./tools/*" = "src/tools/*.gs"

[imports]
"#internal/*" = "src/internal/*.gs"

[dependencies]
"@agent/core" = "workspace:scripts/agent/core"
"toml-tools" = "file:vendor/toml-tools"

[bundle]
target = "dist/hello-agent.bundle.gs"
format = "iife"
includeStd = false
external = ["@std/*"]
```

字段说明：

- `[project]`：保留当前 CLI 项目信息。
- `[package]`：
  - `name`：包名，建议带 scope，如 `@agent/core`、`@local/foo`。
  - `version`：语义版本。
  - `type`：`script` 或 `library`，默认 `script`。
  - `main`：包默认入口，默认沿用 `[project].entry`。
- `[exports]`：包对外可引用入口。未声明时默认 `"." = package.main`。
- `[imports]`：包内部私有别名，建议使用 `#` 前缀，避免和外部包冲突。
- `[dependencies]`：依赖来源。第一阶段支持 `workspace:`、`file:`，远程 registry 先不做。
- `[bundle]`：打包配置。

## 5. 引用协议

推荐将引用字符串分为 6 类，按顺序解析：

1. **native**：`@std/*`
   - 由 `module.GetNative` 处理。
   - 永远不落到文件系统。
2. **relative / absolute**：`./x`、`../x`、`/abs/x`、Windows 绝对路径
   - 相对当前模块目录解析。
   - 默认扩展：`.gs`。
   - 支持目录入口：`dir/project.toml` 的 package main，或 `dir/index.gs`。
3. **project alias**：`@agent/*`
   - 兼容现有规则，映射到项目根 `scripts/agent/*`。
   - 后续可迁移为 workspace package。
4. **package name**：`@scope/name`、`name`、`@scope/name/subpath`
   - 从当前 package root 开始查找依赖表。
   - 通过依赖包的 `[exports]` 解析 subpath。
5. **internal alias**：`#name`、`#name/*`
   - 仅当前包内部可见，来自 `[imports]`。
6. **scheme**：`pkg:name`、`file:path`、`workspace:path`
   - 清单里使用；源码里第一阶段不建议直接使用。

## 6. Resolver 输出

新增统一解析结果类型，供运行时和打包器共用：

```go
type ResolvedModule struct {
    ID          string // 稳定 cache key
    Kind        ModuleKind
    Specifier   string
    Path        string // source/package file path
    PackageRoot string
    PackageName string
    External    bool
}
```

`Kind` 建议枚举：

- `native`
- `source`
- `package`
- `external`

`ID` 规则：

- native：`native:@std/fs`
- source：`file:E:/repo/src/lib.gs`
- package：`pkg:@agent/core@0.1.0:./events`
- external：`external:...`

运行时 cache 应从“绝对路径字符串”升级为 `ResolvedModule.ID`，但 source module 的 `ID` 仍可兼容旧路径。

## 7. 解析算法

输入：

- `specifier`
- `referrer`：当前模块路径或入口路径
- `projectRoot`
- `ResolverOptions`

步骤：

1. 如果是 `@std/*`，返回 native。
2. 如果是相对/绝对路径：
   - 解析为候选路径。
   - 依次尝试：
     - 原路径
     - `.gs`
     - `.json`（可选，后续）
     - 目录 `project.toml` 的 `[package].main`
     - 目录 `index.gs`
3. 如果是 `@agent/*`，先走兼容映射，后续可由 workspace dependency 替代。
4. 如果是 `#` alias：
   - 找到当前 package root。
   - 匹配 `[imports]`。
   - 转换为包内 source module。
5. 如果是包名：
   - 找当前 package root。
   - 读 `gsp.lock` 或 `[dependencies]`。
   - 找依赖 package root。
   - 根据依赖包 `[exports]` 匹配 subpath。
6. 未命中则返回明确错误：
   - `ModuleNotFound`
   - `PackageExportNotFound`
   - `PackageImportNotFound`
   - `PackageManifestInvalid`

## 8. 包导出匹配

导出映射示例：

```toml
[exports]
"." = "src/index.gs"
"./events" = "src/events.gs"
"./tools/*" = "src/tools/*.gs"
```

解析：

- `require("@agent/core")` → `"."`
- `require("@agent/core/events")` → `"./events"`
- `require("@agent/core/tools/bash")` → `"./tools/*"` 替换为 `src/tools/bash.gs`

限制：

- 未在 `[exports]` 声明的包内文件不可被外部引用。
- 包内部可以通过相对路径或 `[imports]` 访问私有文件。

## 9. 依赖锁文件

建议新增 `gsp.lock`，格式可先用 TOML：

```toml
version = 1

[[package]]
name = "@agent/core"
version = "0.1.0"
source = "workspace:scripts/agent/core"
root = "scripts/agent/core"
integrity = ""

[[package]]
name = "toml-tools"
version = "0.2.1"
source = "file:vendor/toml-tools"
root = "vendor/toml-tools"
integrity = "sha256:..."
```

第一阶段只写本地路径和 workspace 依赖。远程 registry、下载缓存和 integrity 校验可以后置。

## 10. 打包设计

打包器不再做简单字符串替换，而是走统一 resolver，构建模块图。

### 10.1 模块图节点

```go
type ModuleNode struct {
    Resolved ResolvedModule
    Source   string
    Imports  []ImportEdge
}
```

`ImportEdge` 记录：

- specifier
- resolved id
- import kind：`require`、`import`、`dynamic`
- source position

### 10.2 产物格式

第一阶段建议支持 `format = "iife"` 的单文件脚本：

```javascript
let __gsp_modules = {};
let __gsp_cache = {};

function __gsp_define(id, factory) {
  __gsp_modules[id] = factory;
}

function __gsp_require(id) {
  if (__gsp_cache[id]) return __gsp_cache[id].exports;
  let module = { exports: {} };
  __gsp_cache[id] = module;
  __gsp_modules[id](module, module.exports, __gsp_require);
  return module.exports;
}

__gsp_define("file:src/lib.gs", function(module, exports, require) {
  // transformed module body
});

__gsp_require("file:src/main.gs");
```

注意：

- native module 默认 external，不打进 bundle。
- `@std/*` 在运行 bundle 时仍通过宿主 native registry 解析。
- 动态 `require(expr)` 第一阶段保留为外部 require，打包器给出 warning。

### 10.3 包归档

后续支持 `.gspkg`：

```text
hello-agent-0.1.0.gspkg
  package.toml
  gsp.lock
  src/...
  dist/...
```

`.gspkg` 可以先定义为 zip。运行时可挂载为 virtual package root，resolver 从归档内解析 source module。

## 11. CLI 命令

建议阶段性增加：

```text
gs run                 # 当前已有 project.toml entry
gs mod init            # 创建 project.toml
gs mod graph           # 打印解析后的模块图
gs mod check           # 校验 manifest、exports、依赖和入口
gs bundle [entry]      # 输出单文件 bundle
gs pack                # 生成 .gspkg
```

第一阶段最小实现：

1. `gs mod graph`
2. `gs bundle`

原因：这两个命令可以验证 resolver 和打包器，不需要先做远程包管理。

## 12. 运行时加载行为

运行时 loader 从当前：

```text
require(path) -> GetNative(path) -> ResolvePath(path, baseDir) -> Cache[absPath]
```

升级为：

```text
require(specifier)
  -> resolver.Resolve(specifier, referrer)
  -> native factory 或 source loader
  -> Cache[ResolvedModule.ID]
```

模块环境仍使用当前 VM：

- 包内所有模块共享同一个 VM。
- 不同 VM 拥有独立 cache、object manager、async state。
- native module factory 是否每次返回新对象保持当前行为；如需单例，应显式在 VM cache 中缓存。

## 13. 与 import/export 的关系

`require` 和 ES `import` 共享 resolver。

- `require("pkg")` 返回 module exports。
- `import x from "pkg"` 取默认导出。
- `import { a } from "pkg"` 取命名导出。
- `import * as ns from "pkg"` 返回 exports namespace。

CommonJS 兼容：

- `exports.foo = value`
- `module.exports = value`

当前 `GetExports(env)` 只读取 `exports`，后续需要同步处理 `module.exports` 重新赋值。

## 14. 实施阶段

### 阶段 A：解析器抽象

- 新增 `internal/module/resolver.go`。
- 定义 `ResolvedModule`、`Resolver`、`ResolveOptions`。
- 保留 `ResolvePath` 作为兼容包装。
- 添加测试：
  - relative
  - absolute
  - `@std/*`
  - `@agent/*`
  - directory main/index

### 阶段 B：manifest 读取

- 扩展 `internal/proj`，支持：
  - `[package]`
  - `[exports]`
  - `[imports]`
  - `[dependencies]`
  - `[bundle]`
- 不强求完整 TOML parser；项目已有 `@std/toml`，Go 端可先实现必要子集，或引入统一 TOML 解析层。

### 阶段 C：运行时接入

- `runner.requireFunc(baseDir)` 改为传入 `referrer`。
- cache key 使用 `ResolvedModule.ID`。
- native/source/package module 共用 loader。
- 加测试覆盖包名 require 和 exports 映射。

### 阶段 D：打包器重写

- `internal/bundle` 使用 resolver 构建模块图。
- 支持 `require("literal")` 和静态 `import`。
- 对动态 require 输出 warning。
- 生成 IIFE bundle。

### 阶段 E：lock 与 pack

- 生成 `gsp.lock`。
- `gs pack` 输出 `.gspkg`。
- 支持 `file:`、`workspace:`。
- 远程 registry 后置。

## 15. 推荐优先级

1. 先实现 resolver 抽象和测试。
2. 再扩展 manifest，不急着做远程依赖。
3. 将运行时 loader 迁到 resolver。
4. 重写 bundle，避免当前字符串替换继续扩大。
5. 最后做 `gsp.lock` 和 `.gspkg`。

这样可以让每一步都能单独合并，而且不会破坏当前脚本运行能力。
