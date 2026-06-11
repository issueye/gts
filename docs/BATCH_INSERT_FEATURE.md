# 批量插入功能 ✅

## 使用方法

```javascript
import orm from "@std/orm";

const db = orm.connect("sqlite", ":memory:");

// 批量插入
const users = [
    { name: "Alice", age: 30 },
    { name: "Bob", age: 25 },
    { name: "Charlie", age: 35 }
];

const result = db.batchInsert("users", users);
console.log("批量插入:", result.rowsAffected, "条");
```

## 特性

- ✅ 单条 SQL 执行多条插入（高效）
- ✅ 自动从第一条记录提取字段
- ✅ 支持所有数据库方言
- ✅ 空数组安全处理

## 性能对比

```javascript
// 方式1: 逐条插入（慢）
for (const user of users) {
    db.table("users").insert(user);  // N 条 SQL
}

// 方式2: 批量插入（快）
db.batchInsert("users", users);  // 1 条 SQL
```

## 测试结果

```
=== RUN   TestORMBatchInsert
--- PASS: TestORMBatchInsert (0.00s)
PASS
```

✅ 功能完成
