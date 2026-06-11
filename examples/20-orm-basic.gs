// ORM 库示例
import orm from "@std/orm";

// 1. 连接数据库
const db = orm.connect("sqlite", ":memory:");

// 创建表
db.table("users").insert({
    name: "Alice",
    email: "alice@example.com",
    age: 30
});

// 2. 查询单条记录
const user = db.table("users")
    .where("email = ?", "alice@example.com")
    .first();

console.log("找到用户:", user);

// 3. 查询多条记录
db.table("users").insert({ name: "Bob", email: "bob@example.com", age: 25 });
db.table("users").insert({ name: "Charlie", email: "charlie@example.com", age: 35 });

const users = db.table("users")
    .where("age >= ?", 25)
    .orderBy("age ASC")
    .find();

console.log("找到", users.length, "个用户");

// 4. 条件查询
const adults = db.table("users")
    .where("age >= ?", 18)
    .where("age < ?", 60)
    .find();

console.log("成年用户:", adults.length);

// 5. 分页查询
const page1 = db.table("users")
    .orderBy("id ASC")
    .limit(2)
    .offset(0)
    .find();

console.log("第一页:", page1);

// 6. 选择特定字段
const names = db.table("users")
    .select("name", "email")
    .find();

console.log("用户名列表:", names);

// 7. 统计
const count = db.table("users").count();
console.log("总用户数:", count);

const adultCount = db.table("users")
    .where("age >= ?", 18)
    .count();
console.log("成年用户数:", adultCount);

// 8. 更新
const updateResult = db.table("users")
    .where("email = ?", "alice@example.com")
    .update({ age: 31 });

console.log("更新了", updateResult.rowsAffected, "条记录");

// 9. 删除
const deleteResult = db.table("users")
    .where("age > ?", 50)
    .delete();

console.log("删除了", deleteResult.rowsAffected, "条记录");

// 10. 事务
const tx = db.begin();

try {
    tx.table("users").insert({
        name: "David",
        email: "david@example.com",
        age: 28
    });

    tx.table("users")
        .where("name = ?", "David")
        .update({ age: 29 });

    tx.commit();
    console.log("事务提交成功");
} catch (err) {
    tx.rollback();
    console.error("事务回滚:", err);
}

// 11. 链式调用
const result = db.table("users")
    .select("name", "age")
    .where("age >= ?", 20)
    .where("age <= ?", 40)
    .orderBy("age DESC")
    .limit(10)
    .find();

console.log("链式查询结果:", result);

// 关闭连接
db.close();
