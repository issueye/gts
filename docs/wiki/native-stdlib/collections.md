# `@std/collections` - 集合操作工具

> 提供常用的集合操作函数，包括分组、去重、分块、集合运算等。

---

## 基础用法

```javascript
let col = require("@std/collections");

// 去重
let unique = col.unique([1, 2, 2, 3, 3, 4]);

// 分组
let grouped = col.groupBy(users, u => u.role);

// 分块
let chunks = col.chunk([1, 2, 3, 4, 5], 2);
```

---

## 分组操作

### groupBy

```javascript
let users = [
  { name: "Alice", role: "admin" },
  { name: "Bob", role: "user" },
  { name: "Charlie", role: "admin" }
];

// 使用函数
let byRole = col.groupBy(users, u => u.role);
// { admin: [{name: "Alice", ...}, {name: "Charlie", ...}], user: [{name: "Bob", ...}] }

// 使用属性名
let byRole = col.groupBy(users, "role");
```

### countBy

```javascript
let items = ["a", "b", "a", "c", "b", "a"];
let counts = col.countBy(items, x => x);
// { a: 3, b: 2, c: 1 }
```

### keyBy

```javascript
let users = [
  { id: 1, name: "Alice" },
  { id: 2, name: "Bob" }
];

let byId = col.keyBy(users, u => u.id);
// { "1": {id: 1, name: "Alice"}, "2": {id: 2, name: "Bob"} }
```

---

## 去重操作

### unique

```javascript
// 基础去重
col.unique([1, 2, 2, 3, 3, 4]);
// [1, 2, 3, 4]

col.unique(["a", "b", "a", "c"]);
// ["a", "b", "c"]
```

### uniqueBy

```javascript
let items = [
  { id: 1, name: "Alice" },
  { id: 2, name: "Bob" },
  { id: 1, name: "Alice2" }
];

// 使用函数
col.uniqueBy(items, item => item.id);
// [{id: 1, name: "Alice"}, {id: 2, name: "Bob"}]

// 使用属性名
col.uniqueBy(items, "id");
```

---

## 分块操作

### chunk

```javascript
col.chunk([1, 2, 3, 4, 5], 2);
// [[1, 2], [3, 4], [5]]

col.chunk([1, 2, 3, 4, 5, 6], 3);
// [[1, 2, 3], [4, 5, 6]]

// 空数组
col.chunk([], 2);
// []
```

---

## 集合运算

### difference

```javascript
// 差集：在第一个数组但不在第二个
col.difference([1, 2, 3], [2, 3, 4]);
// [1]

col.difference([1, 2, 3, 4], [3, 4, 5, 6]);
// [1, 2]
```

### intersection

```javascript
// 交集：同时在两个数组中
col.intersection([1, 2, 3], [2, 3, 4]);
// [2, 3]

col.intersection([1, 2], [3, 4]);
// []
```

### union

```javascript
// 并集：两个数组的所有唯一元素
col.union([1, 2], [2, 3]);
// [1, 2, 3]

col.union([1, 2, 3], [3, 4, 5]);
// [1, 2, 3, 4, 5]
```

---

## 扁平化操作

### flatten

```javascript
// 扁平化一层
col.flatten([[1, 2], [3, 4]]);
// [1, 2, 3, 4]

col.flatten([[1, 2], [3, [4, 5]]]);
// [1, 2, 3, [4, 5]]
```

### flattenDeep

```javascript
// 深度扁平化
col.flattenDeep([[1, [2, [3, [4]]]]]);
// [1, 2, 3, 4]

col.flattenDeep([[1, 2], [3, [4, [5, 6]]]]);
// [1, 2, 3, 4, 5, 6]
```

---

## 分区操作

### partition

```javascript
let nums = [1, 2, 3, 4, 5, 6];
let [evens, odds] = col.partition(nums, n => n % 2 === 0);
// evens: [2, 4, 6]
// odds: [1, 3, 5]

let users = [
  { name: "Alice", active: true },
  { name: "Bob", active: false },
  { name: "Charlie", active: true }
];

let [active, inactive] = col.partition(users, u => u.active);
```

---

## 排序操作

### sortBy

```javascript
let users = [
  { name: "Charlie", age: 30 },
  { name: "Alice", age: 25 },
  { name: "Bob", age: 25 }
];

// 按单个属性
col.sortBy(users, u => u.age);
// [{name: "Alice", age: 25}, {name: "Bob", age: 25}, {name: "Charlie", age: 30}]

// 按多个属性
col.sortBy(users, ["age", "name"]);
// [{name: "Alice", age: 25}, {name: "Bob", age: 25}, {name: "Charlie", age: 30}]
```

---

## 随机操作

### sample

```javascript
// 随机选择一个元素
col.sample([1, 2, 3, 4, 5]);
// 3 (随机)
```

### sampleSize

```javascript
// 随机选择 N 个元素
col.sampleSize([1, 2, 3, 4, 5], 3);
// [2, 4, 1] (随机，不重复)
```

### shuffle

```javascript
// 洗牌
col.shuffle([1, 2, 3, 4, 5]);
// [3, 1, 5, 2, 4] (随机顺序)
```

---

## 范围生成

### range

```javascript
// 基础范围
col.range(5);
// [0, 1, 2, 3, 4]

// 指定起止
col.range(2, 6);
// [2, 3, 4, 5]

// 指定步长
col.range(0, 10, 2);
// [0, 2, 4, 6, 8]

// 负数步长
col.range(5, 0, -1);
// [5, 4, 3, 2, 1]
```

---

## 完整示例

### 数据处理管道

```javascript
let col = require("@std/collections");

let orders = [
  { id: 1, user: "Alice", amount: 100, status: "paid" },
  { id: 2, user: "Bob", amount: 200, status: "pending" },
  { id: 3, user: "Alice", amount: 150, status: "paid" },
  { id: 4, user: "Charlie", amount: 300, status: "paid" },
  { id: 5, user: "Bob", amount: 100, status: "paid" }
];

// 1. 按状态分组
let byStatus = col.groupBy(orders, "status");
console.log("已支付订单数:", byStatus.paid.length);

// 2. 按用户统计
let byUser = col.countBy(orders, o => o.user);
console.log("每个用户的订单数:", byUser);

// 3. 提取唯一用户
let users = col.unique(orders.map(o => o.user));
console.log("用户列表:", users);

// 4. 分区处理
let [large, small] = col.partition(orders, o => o.amount >= 200);
console.log("大额订单:", large.length);
console.log("小额订单:", small.length);

// 5. 按金额排序
let sorted = col.sortBy(orders, o => -o.amount);
console.log("最大订单:", sorted[0]);
```

### 数据分析

```javascript
let col = require("@std/collections");

// 生成测试数据
let data = col.range(100).map(i => ({
  id: i,
  value: Math.floor(Math.random() * 1000),
  category: ["A", "B", "C"][Math.floor(Math.random() * 3)]
}));

// 按类别分组
let byCategory = col.groupBy(data, "category");

// 计算每个类别的统计
for (let [category, items] of Object.entries(byCategory)) {
  let values = items.map(x => x.value);
  let sum = values.reduce((a, b) => a + b, 0);
  let avg = sum / values.length;
  
  console.log(`${category}: 数量=${items.length}, 平均值=${avg.toFixed(2)}`);
}

// 找出前 10 大
let top10 = col.sortBy(data, x => -x.value).slice(0, 10);
console.log("Top 10:", top10.map(x => x.value));

// 分块处理
let batches = col.chunk(data, 10);
console.log(`分成 ${batches.length} 批处理`);
```

---

## API 参考

### 分组函数

| 函数 | 签名 | 说明 |
|------|------|------|
| `groupBy(array, fn)` | `(array, function\|string) → object` | 按键分组 |
| `countBy(array, fn)` | `(array, function\|string) → object` | 统计计数 |
| `keyBy(array, fn)` | `(array, function\|string) → object` | 建立索引 |

### 去重函数

| 函数 | 签名 | 说明 |
|------|------|------|
| `unique(array)` | `(array) → array` | 去重 |
| `uniqueBy(array, fn)` | `(array, function\|string) → array` | 按键去重 |

### 分块函数

| 函数 | 签名 | 说明 |
|------|------|------|
| `chunk(array, size)` | `(array, number) → array[]` | 分块 |

### 集合运算

| 函数 | 签名 | 说明 |
|------|------|------|
| `difference(a, b)` | `(array, array) → array` | 差集 |
| `intersection(a, b)` | `(array, array) → array` | 交集 |
| `union(a, b)` | `(array, array) → array` | 并集 |

### 扁平化函数

| 函数 | 签名 | 说明 |
|------|------|------|
| `flatten(array)` | `(array) → array` | 扁平化一层 |
| `flattenDeep(array)` | `(array) → array` | 深度扁平化 |

### 分区函数

| 函数 | 签名 | 说明 |
|------|------|------|
| `partition(array, fn)` | `(array, function) → [array, array]` | 分区 |

### 排序函数

| 函数 | 签名 | 说明 |
|------|------|------|
| `sortBy(array, fn)` | `(array, function\|string\|string[]) → array` | 排序 |

### 随机函数

| 函数 | 签名 | 说明 |
|------|------|------|
| `sample(array)` | `(array) → any` | 随机一个 |
| `sampleSize(array, n)` | `(array, number) → array` | 随机 N 个 |
| `shuffle(array)` | `(array) → array` | 洗牌 |

### 范围函数

| 函数 | 签名 | 说明 |
|------|------|------|
| `range(end)` | `(number) → array` | 0 到 end-1 |
| `range(start, end)` | `(number, number) → array` | start 到 end-1 |
| `range(start, end, step)` | `(number, number, number) → array` | 带步长 |

---

## 实现状态

✅ **已实现**（2026-06-12）

- ✅ 分组操作
- ✅ 去重操作
- ✅ 分块操作
- ✅ 集合运算
- ✅ 扁平化操作
- ✅ 分区操作
- ✅ 排序操作
- ✅ 随机操作
- ✅ 范围生成
