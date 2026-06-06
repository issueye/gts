# @std/db

> 原生标准库接口单元。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/db` | 原生模块路径 |

## 加载

```javascript
let db = require("@std/db");
```

## 接口

| 接口 | 说明 |
|------|------|
| `drivers` | 支持的数据库驱动名列表 |
| `open(driver, dsn) -> conn` | 打开数据库连接 |
| `conn.exec(query, params?)` | 执行 SQL 并返回影响行数等结果 |
| `conn.query(query, params?)` | 查询多行，返回对象数组 |
| `conn.queryOne(query, params?)` | 查询单行，无结果返回 `null` |
| `conn.prepare(query) -> stmt` | 创建预编译语句 |
| `conn.begin() -> tx` | 开启事务 |
| `conn.setMaxOpenConns(count)` | 设置最大打开连接数 |
| `conn.setMaxIdleConns(count)` | 设置最大空闲连接数 |
| `conn.ping()` | 检查连接是否可用 |
| `conn.close()` | 关闭数据库连接 |
| `tx.exec(query, params?)` | 在事务内执行 SQL |
| `tx.query(query, params?)` | 在事务内查询多行 |
| `tx.queryOne(query, params?)` | 在事务内查询单行 |
| `tx.prepare(query) -> stmt` | 在事务内创建预编译语句 |
| `tx.commit()` | 提交事务 |
| `tx.rollback()` | 回滚事务 |
| `stmt.exec(params?)` | 执行预编译语句 |
| `stmt.query(params?)` | 查询预编译语句 |
| `stmt.queryOne(params?)` | 查询预编译语句单行 |
| `stmt.close()` | 关闭预编译语句 |

## 维护来源

- `internal/stdlib/api_docs.go`
- 对应的 `internal/stdlib/*.go` 模块实现
- 相关示例：`examples/16-native-stdlib.gs`、`examples/17-native-stdlib-cookbook.gs`
