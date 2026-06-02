// hello.gs —— 最小示例
let name: string = "GoScript";
console.log("Hello, " + name + "!");

let nums: number[] = [1, 2, 3, 4, 5];
let sum: number = nums.reduce((a: number, b: number) => a + b, 0);
console.log("sum =", sum);
