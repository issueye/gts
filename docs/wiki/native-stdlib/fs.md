# @std/fs

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/fs` | 原生模块路径 |

## 加载

```javascript
let fs = require("@std/fs");
```

## 接口

| 接口 | 说明 |
|------|------|
| `readFileSync(path)` | 读取文件文本 |
| `readTextSync(path)` | `readFileSync` 的文本别名 |
| `writeFileSync(path, data)` | 写入文件 |
| `writeTextSync(path, data)` | `writeFileSync` 的文本别名 |
| `appendFileSync(path, data)` | 追加写入文件 |
| `appendTextSync(path, data)` | `appendFileSync` 的文本别名 |
| `writeFileAtomicSync(path, data)` | 原子写入文件 |
| `existsSync(path)` | 判断路径是否存在 |
| `readdirSync(path, options?)` | 读取目录条目 |
| `walkSync(root, options?)` | 递归遍历目录 |
| `globSync(pattern)` | 按 glob 模式匹配文件 |
| `copyFileSync(from, to)` | 复制文件 |
| `rmSync(path, options?)` | 删除文件或目录 |
| `mkdtempSync(prefix)` | 创建临时目录 |
| `realpathSync(path)` | 返回真实路径 |
| `lstatSync(path)` | 返回路径状态 |
| `mkdirSync(path, options?)` | 创建目录 |
| `statSync(path)` | 返回文件状态 |
| `renameSync(from, to)` | 重命名或移动 |
| `unlinkSync(path)` | 删除文件 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
