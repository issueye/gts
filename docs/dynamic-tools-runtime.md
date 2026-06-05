# 打包后动态工具脚本

本文档描述语言侧提供的运行期外部脚本能力，以及 agent 生成动态工具时推荐采用的脚本侧约定。

## 目标

打包后的 agent 可执行文件仍然是稳定内核，但在使用过程中可以生成新的 GoScript 工具脚本，并立即加载调用这些工具。

适用场景：

- agent 根据当前项目生成一次性或长期复用工具。
- 工具在打包之后才出现，不能预先写入 `.gspkg`。
- 工具需要复用 GoScript 标准库和相对模块拆分能力。

## 语言侧能力

新增标准库模块：

```javascript
let runtime = require("@std/runtime");
```

### runTool

```javascript
let output = runtime.runTool(path, input, options);
```

要求工具脚本导出：

```javascript
exports.run = function(input) {
  return { ok: true, result: input };
};
```

行为：

- 使用独立 VM 执行外部工具脚本。
- 工具脚本的 `require("./x")` 按工具脚本所在目录解析。
- `@std/*` 标准库正常可用。
- 调用 `exports.run(input)` 并返回结果。
- 工具运行失败会返回运行时错误，不污染主 agent VM。

### runScript

```javascript
let exportsObject = runtime.runScript(path, options);
```

用于调试或加载普通外部脚本。它只执行脚本并返回 `exports`，不会强制调用 `exports.run`。

### options

```javascript
{
  cwd: ".",
  argv: ["agent.exe", "tool-name", "--flag"],
  autoMain: false
}
```

- `cwd`：工具执行时的工作目录，默认继承当前进程 cwd。
- `argv`：设置工具内 `@std/process.argv`。
- `autoMain`：`runScript` 执行后是否自动调用 `main()`。

## 推荐动态工具目录

建议 agent 把生成的工具放在固定目录：

```text
.agent/tools/
  summarize_file/
    tool.toml
    main.gs
    README.md
```

`tool.toml` 示例：

```toml
name = "summarize_file"
description = "Summarize a text file"
entry = "main.gs"

[[params]]
name = "path"
type = "string"
required = true
description = "File path to summarize"
```

`main.gs` 示例：

```javascript
let fs = require("@std/fs");

exports.run = function(input) {
  let text = fs.readFileSync(input.path);
  return {
    ok: true,
    result: text.slice(0, 200)
  };
};
```

## Agent 调用示例

```javascript
let path = require("@std/path");
let runtime = require("@std/runtime");

function callGeneratedTool(name, input) {
  let entry = path.join(".agent", "tools", name, "main.gs");
  return runtime.runTool(entry, input, {
    cwd: process.cwd(),
    argv: ["agent", name]
  });
}

let result = callGeneratedTool("summarize_file", {
  path: "README.md"
});
```

## 打包 exe 手动调试外部脚本

打包后的可执行文件可以显式执行外部脚本：

```text
agent.exe run-script .agent/tools/summarize_file/main.gs -- README.md
```

这用于人工调试。agent 在运行中调用动态工具时，应优先使用 `@std/runtime.runTool`。

## 边界

第一版不提供沙箱。动态工具等同本地代码，可以访问当前可用标准库，包括文件系统和进程执行能力。只应运行可信 agent 或用户生成的工具。

后续可以在 `tool.toml` 增加权限字段，例如：

```toml
[permissions]
fs_read = true
fs_write = false
exec = false
net = true
```
