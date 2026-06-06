# @std/mail

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/mail` | 原生模块路径 |

## 加载

```javascript
let mail = require("@std/mail");
```

## 接口

| 接口 | 说明 |
|------|------|
| `parseAddress(address)` | 解析单个邮件地址 |
| `parseAddressList(addresses)` | 解析邮件地址列表 |
| `parseMessage(message)` | 解析邮件消息文本 |
| `formatAddress(address)` | 格式化邮件地址 |
| `formatAddressList(addresses)` | 格式化邮件地址列表 |
| `parseDate(date)` | 解析邮件日期 |
| `formatDate(time?)` | 格式化为邮件日期字符串 |
| `getHeader(headers, name)` | 从邮件头对象读取字段 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
