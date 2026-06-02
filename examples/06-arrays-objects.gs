// ============================================================
// 06-arrays-objects.gs —— 第六步：数组与对象
// ============================================================

// === 数组 ===
console.log("=== 数组操作 ===");

let arr = [1, 2, 3];

// 增删
arr.push(4);            // 末尾添加
arr.unshift(0);         // 开头添加
console.log("push+unshift:", arr);   // [0, 1, 2, 3, 4]

arr.pop();              // 末尾删除
arr.shift();            // 开头删除
console.log("pop+shift:", arr);      // [1, 2, 3]

// 查找
console.log("indexOf(2):", arr.indexOf(2));     // 1
console.log("includes(5):", arr.includes(5));   // false

// 截取与拼接
console.log("slice(1,2):", arr.slice(1, 2));    // [2]
console.log("concat:", arr.concat([4, 5]));     // [1, 2, 3, 4, 5]

// 变换
console.log("join:", arr.join(" - "));          // "1 - 2 - 3"
console.log("reverse:", arr.reverse());         // [3, 2, 1]

// 排序
let nums = [3, 1, 4, 1, 5, 9];
nums.sort((a, b) => a - b);   // 升序 （必须提供比较函数）
console.log("sort:", nums);                    // [1, 1, 3, 4, 5, 9]

// splice: 原地修改
let colors = ["red", "green", "blue"];
colors.splice(1, 1, "yellow", "purple");  // 从位置1删除1个，插入2个
console.log("splice:", colors);  // ["red", "yellow", "purple", "blue"]

// === 对象 ===
console.log("");
console.log("=== 对象操作 ===");

let person = {
  name: "张三",
  age: 30,
  greet() {
    return `我是${this.name}，今年${this.age}岁`;
  },
};

// 属性访问
console.log("name:", person.name);
console.log("name:", person["name"]);

// 动态属性
let key = "age";
console.log("age via key:", person[key]);

// 方法调用
console.log(person.greet());

// 添加/删除属性
person.city = "北京";
console.log("keys:", Object.keys(person));
console.log("values:", Object.values(person));

// 展开运算符
let base = { a: 1, b: 2 };
let extended = { ...base, c: 3 };
console.log("spread:", extended);

// 数组展开
let a = [1, 2];
let b = [3, 4];
let merged = [...a, ...b];
console.log("array spread:", merged);
