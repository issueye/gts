// 测试框架示例
let test = require("@std/test");

// 基础测试
test("should add numbers", () => {
  test.expect(1 + 1).toBe(2);
  test.expect(2 + 3).toBe(5);
});

test("should multiply numbers", () => {
  test.expect(3 * 4).toBe(12);
});

// 测试套件
test.describe("String operations", () => {
  test.it("should concatenate strings", () => {
    test.expect("hello" + " " + "world").toBe("hello world");
  });

  test.it("should get string length", () => {
    test.expect("hello".length).toBe(5);
  });

  test.it("should check string contains", () => {
    test.expect("hello world").toContain("world");
  });
});

// 数组测试
test.describe("Array operations", () => {
  let arr;

  test.beforeEach(() => {
    arr = [1, 2, 3];
  });

  test.it("should push items", () => {
    arr.push(4);
    test.expect(arr).toHaveLength(4);
    test.expect(arr).toContain(4);
  });

  test.it("should pop items", () => {
    let item = arr.pop();
    test.expect(item).toBe(3);
    test.expect(arr).toHaveLength(2);
  });

  test.it("should access elements", () => {
    test.expect(arr[0]).toBe(1);
    test.expect(arr[2]).toBe(3);
  });
});

// 对象测试
test.describe("Object operations", () => {
  test.it("should have properties", () => {
    let obj = { name: "John", age: 30 };
    test.expect(obj).toHaveProperty("name");
    test.expect(obj).toHaveProperty("name", "John");
    test.expect(obj).toHaveProperty("age", 30);
  });

  test.it("should compare objects", () => {
    let obj1 = { a: 1, b: 2 };
    let obj2 = { a: 1, b: 2 };
    test.expect(obj1).toEqual(obj2);
  });
});

// 真值测试
test.describe("Truthy/Falsy", () => {
  test.it("should test truthy values", () => {
    test.expect(true).toBeTruthy();
    test.expect(1).toBeTruthy();
    test.expect("hello").toBeTruthy();
    test.expect([1, 2]).toBeTruthy();
  });

  test.it("should test falsy values", () => {
    test.expect(false).toBeFalsy();
    test.expect(0).toBeFalsy();
    test.expect("").toBeFalsy();
    test.expect(null).toBeFalsy();
  });
});

// 数字比较
test.describe("Number comparisons", () => {
  test.it("should compare numbers", () => {
    test.expect(10).toBeGreaterThan(5);
    test.expect(10).toBeGreaterThanOrEqual(10);
    test.expect(5).toBeLessThan(10);
    test.expect(5).toBeLessThanOrEqual(5);
  });
});

// 正则匹配
test.describe("Pattern matching", () => {
  test.it("should match patterns", () => {
    test.expect("hello world").toMatch(/world/);
    test.expect("test@example.com").toMatch(/^[\w-\.]+@/);
  });
});

// 否定断言
test.describe("Negation", () => {
  test.it("should negate assertions", () => {
    test.expect(5).not.toBe(10);
    test.expect("hello").not.toContain("world");
    test.expect(true).not.toBeFalsy();
  });
});

// 异常测试
test.describe("Error handling", () => {
  test.it("should throw errors", () => {
    test.expect(() => {
      throw new Error("boom");
    }).toThrow();
  });

  test.it("should not throw", () => {
    test.expect(() => {
      return 42;
    }).not.toThrow();
  });
});

// 嵌套套件
test.describe("Calculator", () => {
  test.describe("Addition", () => {
    test.it("should add positive numbers", () => {
      test.expect(5 + 3).toBe(8);
    });

    test.it("should add negative numbers", () => {
      test.expect(-5 + (-3)).toBe(-8);
    });
  });

  test.describe("Subtraction", () => {
    test.it("should subtract numbers", () => {
      test.expect(10 - 3).toBe(7);
    });
  });
});

// 跳过测试
test.skip("not ready yet", () => {
  test.expect(false).toBe(true); // 这个不会执行
});

// 配置测试
test.configure({
  timeout: 5000,
  verbose: true
});

// 运行所有测试
console.log("\n=== Running Tests ===\n");
let result = test.run();

console.log("\n=== Test Summary ===");
console.log("Total:  ", result.total);
console.log("Passed: ", result.passed);
console.log("Failed: ", result.failed);
console.log("Skipped:", result.skipped);
console.log("Duration:", result.duration, "ms");
