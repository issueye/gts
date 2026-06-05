# 打包后动态脚本

本文档描述语言侧提供的运行期动态脚本能力。动态工具只是动态脚本的一种约定；同一套能力也可用于插件、策略脚本、项目适配脚本、数据转换脚本和调试脚本。

## 目标

打包后的 agent 可执行文件仍然是稳定内核，但在使用过程中可以生成、保存、加载并调用新的 GoScript 脚本。

适用场景：

- agent 根据当前项目生成项目适配脚本。
- 用户在运行过程中添加本地插件脚本。
- 某些能力在打包后才出现，不能预先写入 `.gspkg`。
- 动态脚本需要复用 GoScript 标准库和相对模块拆分能力。

## 语言侧能力

新增标准库模块：

```javascript
let runtime = require("@std/runtime");
```

### runScript

```javascript
let exportsObject = runtime.runScript(path, options);
```

执行外部脚本并返回它的 `exports`。适合加载普通脚本、检查导出、执行带顶层副作用的脚本。

```javascript
let mod = runtime.runScript(".agent/scripts/profile.gs");
println(mod.name);
```

### callScript

```javascript
let output = runtime.callScript(path, exportName, args, options);
```

执行外部脚本，并调用指定导出函数。

```javascript
let output = runtime.callScript(
  ".agent/scripts/convert.gs",
  "convert",
  [{ from: "a.md", to: "a.txt" }]
);
```

动态脚本示例：

```javascript
exports.convert = function(input) {
  return {
    ok: true,
    from: input.from,
    to: input.to
  };
};
```

### runTool

```javascript
let output = runtime.runTool(path, input, options);
```

这是一个便捷函数，等价于调用脚本的 `exports.run(input)`。它只是动态工具约定，不是动态脚本能力的唯一形态。

```javascript
exports.run = function(input) {
  return { ok: true, result: input };
};
```

## options

```javascript
{
  cwd: ".",
  argv: ["agent.exe", "script-name", "--flag"],
  autoMain: false
}
```

- `cwd`：动态脚本执行时的工作目录，默认继承当前进程 cwd。
- `argv`：设置动态脚本内 `@std/process.argv`。
- `autoMain`：`runScript` 执行后是否自动调用 `main()`。

动态脚本的 `require("./x")` 按脚本文件所在目录解析，`@std/*` 标准库正常可用。动态脚本使用独立 VM 执行，不污染主 agent VM。

## 推荐目录

建议把运行期生成的脚本放在固定目录，但目录名称不绑定工具概念：

```text
.agent/scripts/
  convert_markdown/
    script.toml
    main.gs
    README.md

.agent/tools/
  summarize_file/
    tool.toml
    main.gs
```

`script.toml` 示例：

```toml
name = "convert_markdown"
description = "Convert Markdown into another text format"
entry = "main.gs"

[[exports]]
name = "convert"
description = "Convert one file"
```

`main.gs` 示例：

```javascript
let fs = require("@std/fs");

exports.convert = function(input) {
  let text = fs.readFileSync(input.from);
  fs.writeFileSync(input.to, text);
  return { ok: true, bytes: text.length };
};
```

## Agent 调用示例

```javascript
let path = require("@std/path");
let process = require("@std/process");
let runtime = require("@std/runtime");

function callDynamicScript(name, exportName, args) {
  let entry = path.join(".agent", "scripts", name, "main.gs");
  return runtime.callScript(entry, exportName, args, {
    cwd: process.cwd(),
    argv: ["agent", name, exportName]
  });
}

let result = callDynamicScript("convert_markdown", "convert", [{
  from: "README.md",
  to: "README.txt"
}]);
```

动态工具只是一个更窄的封装：

```javascript
function callGeneratedTool(name, input) {
  let entry = path.join(".agent", "tools", name, "main.gs");
  return runtime.runTool(entry, input, {
    cwd: process.cwd(),
    argv: ["agent", "tool", name]
  });
}
```

## 打包 exe 手动调试外部脚本

打包后的可执行文件可以显式执行外部脚本：

```text
agent.exe run-script .agent/scripts/convert_markdown/main.gs -- README.md
```

这用于人工调试。agent 在运行中调用动态脚本时，优先使用 `@std/runtime.runScript` 或 `@std/runtime.callScript`。

## 边界

第一版不提供沙箱。动态脚本等同本地代码，可以访问当前可用标准库，包括文件系统和进程执行能力。只应运行可信 agent 或用户生成的脚本。

后续可以在 `script.toml` 增加权限字段，例如：

```toml
[permissions]
fs_read = true
fs_write = false
exec = false
net = true
```
