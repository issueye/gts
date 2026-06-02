// ============================================================
// 07-types.gs —— 第七步：可选类型注解
// ============================================================
// 类型注解是可选的，在运行时进行校验

// --- 基础类型注解 ---
console.log("=== 基础类型 ===");

let name: string = "GoScript";
let count: number = 42;
let flag: boolean = true;

console.log("name:", name, "count:", count, "flag:", flag);

// --- 函数参数与返回值类型 ---
console.log("");
console.log("=== 函数类型 ===");

function add(a: number, b: number): number {
  return a + b;
}
console.log("add(1, 2) =", add(1, 2));

// void 表示无返回值
function log(msg: string): void {
  console.log("[LOG]", msg);
}
log("类型注解测试");

// --- 数组类型 ---
console.log("");
console.log("=== 数组类型 ===");

let nums: number[] = [1, 2, 3, 4, 5];
let strs: string[] = ["a", "b", "c"];
console.log("nums:", nums, "strs:", strs);

// --- 对象类型（结构类型） ---
console.log("");
console.log("=== 对象类型 ===");

type User = {
  id: number,
  name: string,
  email: string,
};

let alice: User = {
  id: 1,
  name: "Alice",
  email: "alice@example.com",
};
console.log("User:", alice.name, alice.email);

// --- 联合类型 ---
console.log("");
console.log("=== 联合类型 ===");

function stringOrNumber(val: string | number): string {
  if (typeof val === "number") {
    return `number: ${val}`;
  }
  return `string: ${val}`;
}
console.log(stringOrNumber(42));
console.log(stringOrNumber("hello"));

// --- nullable 类型 ---
function findUser(id: number): User | null {
  // 模拟查找
  if (id === 1) {
    return { id: 1, name: "Bob", email: "bob@test.com" };
  }
  return null;
}

let user = findUser(1);
if (user !== null) {
  console.log("找到:", user.name);
} else {
  console.log("未找到");
}

// --- 函数类型参数 ---
console.log("");
console.log("=== 回调类型 ===");

function mapStrings(items: string[], fn: (s: string) => string): string[] {
  return items.map(fn);
}
let upper = mapStrings(["foo", "bar"], s => s.toUpperCase());
console.log("upper:", upper);
