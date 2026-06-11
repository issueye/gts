# API 重命名：create → insert ✅

## 更改说明

为避免歧义，将所有 `create` 方法重命名为 `insert`：

### 之前
```javascript
db.table("users").create({ name: "Alice" });
```

### 之后
```javascript
db.table("users").insert({ name: "Alice" });
```

## 理由

- `insert` 更符合 SQL 标准术语
- 避免与 HTTP 的 CREATE (POST) 概念混淆
- 与其他主流 ORM 保持一致（如 Eloquent 的 `insert()`）

## 影响范围

- ✅ `internal/stdlib/orm.go` - 函数重命名
- ✅ `internal/stdlib/orm_test.go` - 测试更新
- ✅ `docs/orm.md` - 文档更新
- ✅ `docs/ORM_DESIGN_SUMMARY.md` - 设计文档更新
- ✅ `examples/*.gs` - 所有示例更新

## 测试结果

```
=== RUN   TestORMConnect
--- PASS: TestORMConnect (0.00s)
=== RUN   TestORMBasicOperations
--- PASS: TestORMBasicOperations (0.00s)
=== RUN   TestORMQueryOperations
--- PASS: TestORMQueryOperations (0.00s)
=== RUN   TestORMWhereClause
--- PASS: TestORMWhereClause (0.00s)
=== RUN   TestORMWhereIn
--- PASS: TestORMWhereIn (0.00s)
PASS
```

✅ 所有测试通过
