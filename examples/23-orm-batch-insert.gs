// 批量插入示例
import orm from "@std/orm";

const db = orm.connect("sqlite", ":memory:");

// 创建表
db.table("users").insert({ name: "Test", age: 0 });

// 批量插入
const users = [
    { name: "Alice", age: 30 },
    { name: "Bob", age: 25 },
    { name: "Charlie", age: 35 },
    { name: "David", age: 28 },
    { name: "Eve", age: 22 }
];

const result = db.batchInsert("users", users);
console.log("批量插入:", result.rowsAffected, "条记录");

// 验证
const count = db.table("users").count();
console.log("总记录数:", count);

const all = db.table("users").orderBy("name ASC").find();
console.log("所有用户:", all.map(u => u.name));

db.close();
