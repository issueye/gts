// 集合操作示例
let col = require("@std/collections");

console.log("=== 集合操作示例 ===\n");

// 1. 分组操作
console.log("--- 分组操作 ---");
let users = [
  { name: "Alice", role: "admin", age: 30 },
  { name: "Bob", role: "user", age: 25 },
  { name: "Charlie", role: "admin", age: 28 },
  { name: "David", role: "user", age: 35 }
];

let byRole = col.groupBy(users, "role");
console.log("按角色分组:");
console.log("  - admin:", byRole.admin.length, "人");
console.log("  - user:", byRole.user.length, "人");

let counts = col.countBy(users, u => u.role);
console.log("角色计数:", JSON.stringify(counts));

let byId = col.keyBy(users, u => u.name);
console.log("按名字索引:", byId.Alice ? "✓" : "✗");

// 2. 去重操作
console.log("\n--- 去重操作 ---");
let nums = [1, 2, 2, 3, 3, 4, 5, 5];
console.log("原数组:", JSON.stringify(nums));
console.log("去重后:", JSON.stringify(col.unique(nums)));

let items = [
  { id: 1, name: "A" },
  { id: 2, name: "B" },
  { id: 1, name: "A2" }
];
let uniqueItems = col.uniqueBy(items, "id");
console.log("按 id 去重:", uniqueItems.length, "项");

// 3. 分块操作
console.log("\n--- 分块操作 ---");
let data = [1, 2, 3, 4, 5, 6, 7, 8, 9];
let chunks = col.chunk(data, 3);
console.log("分块结果:");
for (let i = 0; i < chunks.length; i++) {
  console.log("  块", i + 1, ":", JSON.stringify(chunks[i]));
}

// 4. 集合运算
console.log("\n--- 集合运算 ---");
let set1 = [1, 2, 3, 4];
let set2 = [3, 4, 5, 6];
console.log("集合 1:", JSON.stringify(set1));
console.log("集合 2:", JSON.stringify(set2));
console.log("差集 (1-2):", JSON.stringify(col.difference(set1, set2)));
console.log("交集:", JSON.stringify(col.intersection(set1, set2)));
console.log("并集:", JSON.stringify(col.union(set1, set2)));

// 5. 扁平化操作
console.log("\n--- 扁平化操作 ---");
let nested1 = [[1, 2], [3, 4], [5]];
console.log("一层嵌套:", JSON.stringify(nested1));
console.log("扁平化:", JSON.stringify(col.flatten(nested1)));

let nested2 = [[1, [2, [3, [4]]]]];
console.log("深层嵌套:", JSON.stringify(nested2));
console.log("深度扁平化:", JSON.stringify(col.flattenDeep(nested2)));

// 6. 分区操作
console.log("\n--- 分区操作 ---");
let numbers = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];
let parts = col.partition(numbers, n => n % 2 === 0);
console.log("原数组:", JSON.stringify(numbers));
console.log("偶数:", JSON.stringify(parts[0]));
console.log("奇数:", JSON.stringify(parts[1]));

let activeUsers = col.partition(users, u => u.age >= 30);
console.log("30+ 岁:", activeUsers[0].length, "人");
console.log("30 岁以下:", activeUsers[1].length, "人");

// 7. 排序操作
console.log("\n--- 排序操作 ---");
let unsorted = [
  { name: "Charlie", score: 85 },
  { name: "Alice", score: 95 },
  { name: "Bob", score: 85 }
];
let sorted = col.sortBy(unsorted, u => u.score);
console.log("按分数排序:");
for (let i = 0; i < sorted.length; i++) {
  console.log("  -", sorted[i].name, ":", sorted[i].score);
}

let multiSort = col.sortBy(unsorted, ["score", "name"]);
console.log("按分数和名字排序:");
for (let i = 0; i < multiSort.length; i++) {
  console.log("  -", multiSort[i].name, ":", multiSort[i].score);
}

// 8. 随机操作
console.log("\n--- 随机操作 ---");
let pool = [1, 2, 3, 4, 5];
console.log("数组:", JSON.stringify(pool));
console.log("随机一个:", col.sample(pool));
console.log("随机三个:", JSON.stringify(col.sampleSize(pool, 3)));
console.log("洗牌:", JSON.stringify(col.shuffle(pool)));

// 9. 范围生成
console.log("\n--- 范围生成 ---");
console.log("range(5):", JSON.stringify(col.range(5)));
console.log("range(2, 6):", JSON.stringify(col.range(2, 6)));
console.log("range(0, 10, 2):", JSON.stringify(col.range(0, 10, 2)));
console.log("range(5, 0, -1):", JSON.stringify(col.range(5, 0, -1)));

// 10. 实际应用示例
console.log("\n--- 实际应用：订单分析 ---");
let orders = [
  { id: 1, user: "Alice", amount: 100, status: "paid" },
  { id: 2, user: "Bob", amount: 200, status: "pending" },
  { id: 3, user: "Alice", amount: 150, status: "paid" },
  { id: 4, user: "Charlie", amount: 300, status: "paid" },
  { id: 5, user: "Bob", amount: 100, status: "paid" },
  { id: 6, user: "Alice", amount: 50, status: "pending" }
];

// 按状态分组
let byStatus = col.groupBy(orders, "status");
console.log("订单状态:");
console.log("  - 已支付:", byStatus.paid.length, "单");
console.log("  - 待支付:", byStatus.pending.length, "单");

// 按用户统计
let byUser = col.countBy(orders, o => o.user);
console.log("用户订单数:");
for (let user in byUser) {
  console.log("  -", user, ":", byUser[user], "单");
}

// 提取唯一用户
let uniqueUsers = col.unique(orders.map(o => o.user));
console.log("用户列表:", JSON.stringify(uniqueUsers));

// 大额订单（金额 >= 200）
let largeOrders = col.partition(orders, o => o.amount >= 200);
console.log("大额订单:", largeOrders[0].length, "单");
console.log("小额订单:", largeOrders[1].length, "单");

// 按金额排序
let sortedOrders = col.sortBy(orders, o => -o.amount);
console.log("最大订单:");
console.log("  - 用户:", sortedOrders[0].user);
console.log("  - 金额:", sortedOrders[0].amount);

// 分批处理
let batches = col.chunk(orders, 3);
console.log("分", batches.length, "批处理");

console.log("\n✓ 集合操作示例完成");
