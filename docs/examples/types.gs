// types.gs —— 类型注解的使用与运行时检查
function add(a: number, b: number): number {
  return a + b;
}

function findById(items: { id: number, name: string }[], id: number): { id: number, name: string } | null {
  for (let item of items) {
    if (item.id === id) return item;
  }
  return null;
}

let users: { id: number, name: string }[] = [
  { id: 1, name: "Alice" },
  { id: 2, name: "Bob" },
];

console.log(add(1, 2));            // 3
console.log(findById(users, 2));   // { id: 2, name: "Bob" }
