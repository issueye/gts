// ============================================================
// 10-classes.gs —— 第十步：类与继承
// ============================================================

// --- 基础类 ---
console.log("=== 基础类 ===");

class Point {
  x: number;
  y: number;

  constructor(x: number, y: number) {
    this.x = x;
    this.y = y;
  }

  distance(): number {
    return Math.sqrt(this.x * this.x + this.y * this.y);
  }

  toString(): string {
    return `Point(${this.x}, ${this.y})`;
  }
}

let p = new Point(3, 4);
console.log(p.toString());
console.log("distance =", p.distance());   // 5

// --- 继承 ---
console.log("");
console.log("=== 继承 ===");

class Point3D extends Point {
  z: number;

  constructor(x: number, y: number, z: number) {
    super(x, y);          // 调用父类构造器
    this.z = z;
  }

  // 重写父类方法
  distance(): number {
    return Math.sqrt(this.x * this.x + this.y * this.y + this.z * this.z);
  }

  toString(): string {
    return `Point3D(${this.x}, ${this.y}, ${this.z})`;
  }
}

let p3 = new Point3D(3, 4, 12);
console.log(p3.toString());
console.log("distance =", p3.distance());  // 13

// --- 多态 ---
console.log("");
console.log("=== 多态 ===");

class Animal {
  name: string;

  constructor(name: string) {
    this.name = name;
  }

  speak(): string {
    return `${this.name} 发出声音`;
  }
}

class Dog extends Animal {
  speak(): string {
    return `${this.name} 汪汪叫`;
  }
}

class Cat extends Animal {
  speak(): string {
    return `${this.name} 喵喵叫`;
  }
}

// 多态：同一个方法，不同行为
let animals: Animal[] = [
  new Animal("未知生物"),
  new Dog("旺财"),
  new Cat("咪咪"),
];

for (let a of animals) {
  console.log(a.speak());
}

// --- instanceof 检查 ---
console.log("");
console.log("=== instanceof ===");
let dog = new Dog("小黑");
console.log("dog instanceof Dog:", dog instanceof Dog);        // true
console.log("dog instanceof Animal:", dog instanceof Animal);  // true
console.log("dog instanceof Cat:", dog instanceof Cat);        // false

// --- 静态方法 ---
console.log("");
console.log("=== 静态方法 ===");

class Calculator {
  static add(a: number, b: number): number {
    return a + b;
  }

  static PI: number = 3.1415926;
}
console.log("Calculator.add(1,2) =", Calculator.add(1, 2));
console.log("Calculator.PI =", Calculator.PI);
