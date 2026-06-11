// WHERE IN 示例
import orm from "@std/orm";

const db = orm.connect("sqlite", ":memory:");

// 创建表
db.table("users").insert({ name: "Alice", age: 30, city: "Beijing" });
db.table("users").insert({ name: "Bob", age: 25, city: "Shanghai" });
db.table("users").insert({ name: "Charlie", age: 35, city: "Guangzhou" });
db.table("users").insert({ name: "David", age: 28, city: "Beijing" });

// 方式 1: where + 数组参数（自动展开）
const users1 = db.table("users")
    .where("city IN (?, ?)", ["Beijing", "Shanghai"])
    .find();

console.log("方式1 - where + 数组:", users1.length); // 3

// 方式 2: whereIn 辅助方法（推荐）
const users2 = db.table("users")
    .whereIn("city", ["Beijing", "Shanghai"])
    .find();

console.log("方式2 - whereIn:", users2.length); // 3

// 结合其他条件
const users3 = db.table("users")
    .whereIn("city", ["Beijing", "Shanghai"])
    .where("age >= ?", 28)
    .find();

console.log("结合条件:", users3.length); // 2 (David 和 Alice)

// ID 列表查询
db.table("users").insert({ name: "Eve", age: 22, city: "Shenzhen" });

const users4 = db.table("users")
    .whereIn("id", [1, 2, 5])
    .orderBy("id ASC")
    .find();

console.log("ID 列表:", users4.map(u => u.name)); // ["Alice", "Bob", "Eve"]

db.close();
