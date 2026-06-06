# @std/url

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/url` | 原生模块路径 |

## 加载

```javascript
let url = require("@std/url");
```

## 接口

| 接口 | 说明 |
|------|------|
| `parse(url) -> URL` | 解析 URL 字符串 |
| `format(urlObject)` | 格式化 URL 对象 |
| `resolve(base, ref)` | 解析相对 URL |
| `pathToFileURL(path)` | 文件路径转 file URL |
| `fileURLToPath(url)` | file URL 转文件路径 |
| `URL(input, base?) -> URL` | 构造 URL 对象 |
| `URLSearchParams(init?) -> params` | 构造查询参数对象 |
| `url.toString()/toJSON()` | 返回 URL 字符串 |
| `params.get(name)/has(name)` | 读取查询参数 |
| `params.set(name, value)/append(name, value)` | 设置或追加查询参数 |
| `params.delete(name)` | 删除查询参数 |
| `params.toString()` | 序列化查询参数 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
