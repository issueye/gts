// errors.gs —— 错误处理
class Validator {
  static require(cond: boolean, msg: string): void {
    if (!cond) throw new RangeError(msg);
  }
}

function divide(a: number, b: number): number {
  try {
    Validator.require(b !== 0, "divide by zero");
    return a / b;
  } catch (e) {
    console.error("caught:", e.name, e.message);
    throw e; // 重新抛出
  } finally {
    console.log("divide cleanup");
  }
}

try {
  console.log(divide(10, 0));
} catch (e) {
  console.error("outer caught:", e.message);
}
