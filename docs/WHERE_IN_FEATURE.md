# WHERE IN 功能已添加 ✅

## 两种用法

### 1. where() + 数组参数（自动展开）
```javascript
db.table("users")
    .where("city IN (?, ?)", ["Beijing", "Shanghai"])
    .find();
```

### 2. whereIn() 辅助方法（推荐）
```javascript
db.table("users")
    .whereIn("city", ["Beijing", "Shanghai"])
    .find();

// 支持任意数量的值
db.table("users")
    .whereIn("id", [1, 2, 3, 5, 8])
    .find();
```

## 实现细节

- `where()` 方法会自动检测数组参数并展开
- `whereIn()` 方法自动生成正确数量的占位符
- 支持所有数据库（MySQL: `?`, PostgreSQL: `$1, $2`）
- 可以与其他条件链式组合

## 测试结果

```
=== RUN   TestORMWhereIn
--- PASS: TestORMWhereIn (0.00s)
PASS
```

✅ 功能已完成并测试通过
