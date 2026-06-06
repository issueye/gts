# @std/net/ip

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/net/ip` | 原生模块路径 |

## 加载

```javascript
let netIp = require("@std/net/ip");
```

## 接口

| 接口 | 说明 |
|------|------|
| `parseIP(ip)` | 解析 IP 地址 |
| `parseCIDR(cidr)` | 解析 CIDR 网段 |
| `contains(cidr, ip)` | 判断网段是否包含 IP |
| `splitHostPort(address)` | 拆分 `host:port` |
| `joinHostPort(host, port)` | 拼接 `host:port` |
| `lookupHost(host)` | DNS 查询主机地址 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
