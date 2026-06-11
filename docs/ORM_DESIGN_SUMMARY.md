# GTS ORM 系统设计完成

## 项目概述

为 GTS 脚本语言设计并实现了一个原生的 ORM（对象关系映射）库，提供简洁优雅的数据库操作接口。

## 已完成的工作

### 1. 核心文件

#### `internal/stdlib/orm.go` (650+ 行)
完整的 ORM 实现，包括：
- 数据库连接管理
- 链式查询构建器
- CRUD 操作
- 事务支持
- 多数据库方言支持（MySQL, PostgreSQL, SQLite, SQL Server）

#### `internal/stdlib/orm_test.go` (280+ 行)
全面的单元测试：
- 数据库连接测试
- CRUD 操作测试
- 查询构建器测试
- WHERE 条件测试
- **所有测试通过** ✅

### 2. 文档

#### `docs/orm.md` (400+ 行)
详尽的使用文档：
- 快速开始指南
- API 参考
- 实际示例
- 最佳实践
- 与 @std/db 的对比

#### `examples/20-orm-basic.gs` (100+ 行)
基础示例，涵盖：
- 连接数据库
- CRUD 操作
- 查询条件
- 分页
- 事务

#### `examples/21-orm-advanced.gs` (250+ 行)
高级示例，包含：
- 用户管理系统
- 文章管理系统
- 评论系统
- 统计分析
- 批量操作

## 核心特性

### 1. 链式 API

```javascript
const users = db.table("users")
    .select("name", "email")
    .where("age >= ?", 18)
    .where("status = ?", "active")
    .orderBy("created_at DESC")
    .limit(10)
    .offset(20)
    .find();
```

### 2. 多数据库支持

| 数据库 | 驱动名 | 占位符 | 引用符 |
|--------|--------|--------|--------|
| MySQL | `mysql` | `?` | `` ` `` |
| PostgreSQL | `postgres` | `$1, $2` | `"` |
| SQLite | `sqlite` | `?` | `"` |
| SQL Server | `sqlserver` | `?` | `[]` |

### 3. CRUD 操作

```javascript
// Insert
db.table("users").insert({ name: "Alice", age: 30 });

// Read
const user = db.table("users").where("id = ?", 1).first();
const users = db.table("users").where("age >= ?", 18).find();

// Update
db.table("users").where("id = ?", 1).update({ age: 31 });

// Delete
db.table("users").where("id = ?", 1).delete();
```

### 4. 事务支持

```javascript
const tx = db.begin();
try {
    tx.table("accounts").where("id = ?", 1).update({ balance: balance - 100 });
    tx.table("accounts").where("id = ?", 2).update({ balance: balance + 100 });
    tx.commit();
} catch (err) {
    tx.rollback();
}
```

### 5. 查询构建器方法

- `select(...fields)` - 选择字段
- `where(condition, ...args)` - 添加条件
- `orderBy(...fields)` - 排序
- `limit(n)` - 限制数量
- `offset(n)` - 偏移量
- `find()` - 查询多条
- `first()` - 查询单条
- `count()` - 统计数量
- `insert(data)` - 插入记录
- `update(data)` - 更新记录
- `delete()` - 删除记录

## 技术架构

### 核心结构

```
┌─────────────────┐
│  @std/orm       │  ← GTS 模块入口
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  ormDB          │  ← 数据库连接包装
├─────────────────┤
│  - conn         │
│  - table()      │
│  - begin()      │
│  - close()      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  ormModel       │  ← 查询构建器
├─────────────────┤
│  - table        │
│  - fields       │
│  - where        │
│  - orderBy      │
│  - limit/offset │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  SQL Builder    │  ← SQL 生成
├─────────────────┤
│  - buildSelect  │
│  - quote        │
│  - placeholder  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  database/sql   │  ← Go 标准库
└─────────────────┘
```

### 设计模式

1. **构建器模式** - 链式 API 构建查询
2. **策略模式** - 数据库方言支持
3. **装饰器模式** - 对象封装
4. **工厂模式** - 连接创建

## 与现有系统集成

### 与 @std/db 的关系

```javascript
// @std/db - 原始 SQL（更灵活）
import db from "@std/db";
const conn = db.open("mysql", dsn);
const users = conn.query("SELECT * FROM users WHERE age >= ?", [18]);

// @std/orm - 链式 API（更简洁）
import orm from "@std/orm";
const db = orm.connect("mysql", dsn);
const users = db.table("users").where("age >= ?", 18).find();
```

两者可以共存，根据场景选择：
- 简单 CRUD → 使用 ORM
- 复杂查询 → 使用原始 SQL

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
PASS
ok  	github.com/issueye/goscript/internal/stdlib	0.046s
```

**所有测试通过！** ✅

## 使用示例

### 基础示例

```javascript
import orm from "@std/orm";

const db = orm.connect("sqlite", ":memory:");

// 创建
db.table("users").insert({
    name: "Alice",
    email: "alice@example.com",
    age: 30
});

// 查询
const user = db.table("users")
    .where("email = ?", "alice@example.com")
    .first();

console.log(user.name); // "Alice"

// 更新
db.table("users")
    .where("id = ?", user.id)
    .update({ age: 31 });

// 删除
db.table("users")
    .where("id = ?", user.id)
    .delete();

db.close();
```

### 实际应用

```javascript
// 用户注册
function register(email, password) {
    const existing = db.table("users")
        .where("email = ?", email)
        .first();
    
    if (existing) {
        throw new Error("邮箱已被注册");
    }
    
    return db.table("users").insert({
        email: email,
        password: hashPassword(password),
        created_at: Date.now()
    });
}

// 分页查询
function getUsers(page, pageSize) {
    return db.table("users")
        .select("id", "email", "created_at")
        .orderBy("created_at DESC")
        .limit(pageSize)
        .offset((page - 1) * pageSize)
        .find();
}
```

## 性能特点

- **轻量级** - 基于 Go database/sql，无额外依赖
- **高效** - 参数化查询，自动预处理
- **安全** - 防止 SQL 注入
- **灵活** - 支持原始 SQL 和构建器混用

## 后续扩展方向

### 可选增强功能

1. **关联关系**
   - hasOne / hasMany
   - belongsTo / belongsToMany

2. **模型定义**
   - 字段验证
   - 默认值
   - 钩子函数

3. **高级查询**
   - JOIN 支持
   - 子查询
   - 聚合函数

4. **迁移工具**
   - Schema 定义
   - 版本管理
   - 数据迁移

5. **缓存层**
   - 查询缓存
   - 模型缓存

## 总结

成功为 GTS 语言设计并实现了一个功能完整、测试通过的原生 ORM 系统。该系统提供：

✅ 简洁的链式 API  
✅ 多数据库支持  
✅ 完整的 CRUD 操作  
✅ 事务管理  
✅ 全面的文档和示例  
✅ 完整的单元测试覆盖  

开发者现在可以在 GTS 脚本中使用 `import orm from "@std/orm"` 来进行优雅的数据库操作。
