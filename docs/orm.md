# @std/orm - GTS ORM 库

GTS 语言的原生 ORM（对象关系映射）库，提供简洁优雅的数据库操作接口。

## 特性

- ✅ **链式调用** - 流畅的查询构建器 API
- ✅ **多数据库支持** - MySQL、PostgreSQL、SQLite、SQL Server
- ✅ **事务支持** - 完整的事务管理
- ✅ **类型安全** - 参数化查询，防止 SQL 注入
- ✅ **简洁易用** - 零配置，开箱即用

## 快速开始

### 安装

ORM 库是 GTS 标准库的一部分，无需安装。

```javascript
import orm from "@std/orm";
```

### 基础用法

#### 1. 连接数据库

```javascript
// SQLite（内存数据库）
const db = orm.connect("sqlite", ":memory:");

// MySQL
const db = orm.connect("mysql", "user:password@tcp(localhost:3306)/dbname");

// PostgreSQL
const db = orm.connect("postgres", "postgres://user:password@localhost:5432/dbname");

// SQL Server
const db = orm.connect("sqlserver", "sqlserver://user:password@localhost:1433?database=dbname");
```

#### 2. 插入记录

```javascript
// 创建单条记录
const result = db.table("users").insert({
    name: "Alice",
    email: "alice@example.com",
    age: 30
});

console.log("插入 ID:", result.lastInsertId);
console.log("影响行数:", result.rowsAffected);
```

#### 3. 查询记录

```javascript
// 查询单条
const user = db.table("users")
    .where("id = ?", 1)
    .first();

// 查询多条
const users = db.table("users")
    .where("age >= ?", 18)
    .find();

// 选择字段
const names = db.table("users")
    .select("name", "email")
    .find();

// 排序
const sorted = db.table("users")
    .orderBy("age DESC")
    .find();

// 分页
const page = db.table("users")
    .limit(10)
    .offset(20)
    .find();

// 统计
const count = db.table("users")
    .where("status = ?", "active")
    .count();

// WHERE IN
const users = db.table("users")
    .whereIn("city", ["Beijing", "Shanghai", "Guangzhou"])
    .find();
```

#### 4. 更新记录

```javascript
const result = db.table("users")
    .where("id = ?", 1)
    .update({
        age: 31,
        updated_at: Date.now()
    });

console.log("更新了", result.rowsAffected, "条记录");
```

#### 5. 删除记录

```javascript
const result = db.table("users")
    .where("status = ?", "inactive")
    .delete();

console.log("删除了", result.rowsAffected, "条记录");
```

#### 6. 事务

```javascript
const tx = db.begin();

try {
    tx.table("accounts")
        .where("id = ?", 1)
        .update({ balance: balance - 100 });
    
    tx.table("accounts")
        .where("id = ?", 2)
        .update({ balance: balance + 100 });
    
    tx.commit();
    console.log("转账成功");
} catch (err) {
    tx.rollback();
    console.error("转账失败:", err);
}
```

## API 参考

### orm.connect(driver, dsn)

连接数据库。

**参数:**
- `driver` (string) - 数据库驱动名称: `"mysql"`, `"postgres"`, `"sqlite"`, `"sqlserver"`
- `dsn` (string) - 数据源名称（连接字符串）

**返回:** Database 对象

### Database

#### db.table(name)

选择表。

**返回:** QueryBuilder 对象

#### db.begin()

开始事务。

**返回:** Transaction 对象

#### db.close()

关闭数据库连接。

### QueryBuilder

查询构建器支持链式调用。

#### .select(...fields)

选择要返回的字段。

```javascript
db.table("users").select("id", "name", "email")
```

#### .where(condition, ...args)

添加 WHERE 条件。支持数组参数自动展开（用于 IN 查询）。

```javascript
db.table("users").where("age >= ?", 18)
db.table("users").where("age >= ?", 18).where("status = ?", "active")

// WHERE IN - 数组自动展开
db.table("users").where("city IN (?, ?)", ["Beijing", "Shanghai"])
```

#### .whereIn(field, values)

WHERE IN 辅助方法，更简洁的语法。

```javascript
db.table("users").whereIn("city", ["Beijing", "Shanghai"])
db.table("users").whereIn("id", [1, 2, 3, 5])
```

#### .orderBy(...fields)

排序。

```javascript
db.table("users").orderBy("age DESC")
db.table("users").orderBy("age DESC", "name ASC")
```

#### .limit(n)

限制返回数量。

```javascript
db.table("users").limit(10)
```

#### .offset(n)

偏移量（用于分页）。

```javascript
db.table("users").limit(10).offset(20)
```

#### .find()

执行查询，返回多条记录。

**返回:** Array

#### .first()

执行查询，返回第一条记录。

**返回:** Object 或 null

#### .count()

统计记录数。

**返回:** Number

#### .insert(data)

插入记录。

**参数:** Object
**返回:** { lastInsertId, rowsAffected }

#### .update(data)

更新记录。

**参数:** Object
**返回:** { rowsAffected }

#### .delete()

删除记录。

**返回:** { rowsAffected }

### Transaction

事务对象，API 与 Database 相同，但在事务上下文中执行。

#### tx.table(name)

选择表（事务上下文）。

#### tx.commit()

提交事务。

#### tx.rollback()

回滚事务。

## 实际示例

### 用户管理

```javascript
import orm from "@std/orm";

const db = orm.connect("mysql", "root:password@tcp(localhost:3306)/myapp");

// 注册用户
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

// 用户登录
function login(email, password) {
    const user = db.table("users")
        .where("email = ?", email)
        .first();
    
    if (!user || !verifyPassword(password, user.password)) {
        throw new Error("邮箱或密码错误");
    }
    
    return user;
}

// 获取用户列表
function getUsers(page, pageSize) {
    return db.table("users")
        .select("id", "email", "created_at")
        .orderBy("created_at DESC")
        .limit(pageSize)
        .offset((page - 1) * pageSize)
        .find();
}
```

### 电商订单

```javascript
// 创建订单
function createOrder(userId, items) {
    const tx = db.begin();
    
    try {
        // 创建订单
        const orderResult = tx.table("orders").insert({
            user_id: userId,
            total: calculateTotal(items),
            status: "pending",
            created_at: Date.now()
        });
        
        const orderId = orderResult.lastInsertId;
        
        // 创建订单项
        for (const item of items) {
            tx.table("order_items").insert({
                order_id: orderId,
                product_id: item.productId,
                quantity: item.quantity,
                price: item.price
            });
            
            // 减少库存
            tx.table("products")
                .where("id = ?", item.productId)
                .update({
                    stock: stock - item.quantity
                });
        }
        
        tx.commit();
        return orderId;
    } catch (err) {
        tx.rollback();
        throw err;
    }
}
```

## 最佳实践

### 1. 使用参数化查询

```javascript
// ✅ 推荐
db.table("users").where("email = ?", userInput)

// ❌ 不推荐（SQL 注入风险）
db.table("users").where("email = '" + userInput + "'")
```

### 2. 合理使用事务

```javascript
// 多个相关操作应该在事务中执行
const tx = db.begin();
try {
    // 相关操作
    tx.commit();
} catch (err) {
    tx.rollback();
}
```

### 3. 关闭连接

```javascript
// 程序结束前关闭连接
db.close();
```

### 4. 错误处理

```javascript
try {
    const user = db.table("users")
        .where("id = ?", userId)
        .first();
    
    if (!user) {
        console.log("用户不存在");
    }
} catch (err) {
    console.error("数据库错误:", err);
}
```

## 与 @std/db 的区别

| 特性 | @std/db | @std/orm |
|------|---------|----------|
| API 风格 | 原始 SQL | 链式构建器 |
| 易用性 | 中等 | 简单 |
| 灵活性 | 高 | 中等 |
| 适用场景 | 复杂 SQL | 常规 CRUD |

```javascript
// @std/db - 原始 SQL
import db from "@std/db";
const conn = db.open("mysql", dsn);
const users = conn.query("SELECT * FROM users WHERE age >= ?", [18]);

// @std/orm - 链式 API
import orm from "@std/orm";
const db = orm.connect("mysql", dsn);
const users = db.table("users").where("age >= ?", 18).find();
```

## 注意事项

1. **占位符** - PostgreSQL 使用 `$1, $2`，其他数据库使用 `?`（ORM 自动处理）
2. **字段引用** - 表名和字段名会根据数据库类型自动添加引号
3. **类型转换** - 数据会自动在 GTS 类型和 SQL 类型之间转换
4. **性能** - 对于超大数据量和复杂查询，建议使用 `@std/db` 直接写 SQL

## 许可证

MIT
