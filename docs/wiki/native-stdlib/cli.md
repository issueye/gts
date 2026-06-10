# @std/cli

> Cobra 风格的命令行框架原生库。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/cli` | 原生模块路径 |

## 加载

```javascript
let cli = require("@std/cli");
```

## 快速示例

```javascript
let cli = require("@std/cli");

let root = cli.command({
  use: "app",
  short: "示例应用",
  version: "1.0.0",
});

root.persistentFlags().string("profile", "p", "dev", "运行配置");

let serve = root.command({
  use: "serve [dir]",
  short: "启动服务",
  args: cli.exactArgs(1),
  run: function(cmd, args) {
    println("profile=" + cmd.flag("profile"));
    println("port=" + String(cmd.flag("port")));
    println("dir=" + args[0]);
  },
});

serve.flags().int("port", "", 8080, "监听端口");
root.execute();
```

## 接口

| 接口 | 说明 |
|------|------|
| `command(options?) -> cmd` | 创建命令对象 |
| `root(options?) -> cmd` | `command` 的别名，用于创建根命令 |
| `cmd.addCommand(...commands)` | 添加子命令 |
| `cmd.command(options?) -> cmd` | 创建并添加子命令 |
| `cmd.flags() -> flags` | 返回当前命令 flag 集 |
| `cmd.persistentFlags() -> flags` | 返回可被子命令继承的 flag 集 |
| `flags.string(name, shorthand?, default?, usage?)` | 定义字符串 flag |
| `flags.bool(name, shorthand?, default?, usage?)` | 定义布尔 flag |
| `flags.int(name, shorthand?, default?, usage?)` | 定义整数 flag |
| `flags.number(name, shorthand?, default?, usage?)` | 定义数字 flag |
| `flags.get(name)` | 读取当前 flag 集中的 flag 值 |
| `flags.changed(name)` | 判断当前 flag 集中的 flag 是否由命令行显式设置 |
| `cmd.execute(args?)` | 解析参数、匹配子命令并执行回调；不传 args 时使用 `process.argv.slice(1)` |
| `cmd.flag(name)` | 读取当前命令解析后的 flag 值，包含继承的 persistent flag |
| `cmd.usage()` | 返回帮助文本 |
| `cmd.help()` | 输出帮助文本 |
| `cmd.commandPath()` | 返回命令路径 |
| `noArgs()` | 不允许位置参数 |
| `arbitraryArgs()` | 允许任意位置参数 |
| `exactArgs(n)` | 要求恰好 n 个位置参数 |
| `minArgs(n)` | 要求至少 n 个位置参数 |
| `maxArgs(n)` | 要求至多 n 个位置参数 |
| `rangeArgs(min, max)` | 要求位置参数数量在范围内 |

## options 字段

| 字段 | 说明 |
|------|------|
| `use` / `Use` | 命令用法，例如 `serve [dir]` |
| `short` / `Short` | 单行说明 |
| `long` / `Long` | 详细说明 |
| `example` / `Example` | 示例文本 |
| `version` / `Version` | 根命令版本，自动支持 `--version` 和 `-v` |
| `aliases` / `Aliases` | 子命令别名数组 |
| `args` / `Args` | 参数校验器或自定义校验函数 |
| `preRun` / `PreRun` | `run` 前执行的回调 |
| `run` / `Run` | 命令主体回调，签名为 `function(cmd, args)` |
| `postRun` / `PostRun` | `run` 后执行的回调 |

## 参数与 flag

- 长 flag 支持 `--name value` 和 `--name=value`。
- 布尔 flag 支持 `--name`、`--name=true`、`--name=false` 和 `--no-name`。
- 短 flag 支持 `-p value`，布尔短 flag 可组合，例如 `-av`。
- `--help` / `-h` 会输出当前命令帮助并跳过 `run` 回调。

## 维护来源

- `internal/stdlib/cli.go`
- `internal/stdlib/api_docs.go`
- 相关测试：`internal/stdlib/cli_test.go`
