// closure-best-practices.gs - 闭包最佳实践

println("=== GoScript 闭包最佳实践 ===");
println("");

// ❌ 错误示例 1: 循环中直接使用闭包
println("❌ 错误示例 - 循环变量闭包捕获:");
println("(所有 goroutine 捕获同一个变量)");
println("");

// ✅ 正确示例 1: 通过参数传递
println("✅ 正确方式 1 - 参数传递:");
for (let i = 0; i < 5; i++) {
  go(function(val) {
    println("goroutine " + String(val));
  }, i);  // 将 i 的值作为参数传递
}

sleep(100);
println("");

// ✅ 正确示例 2: 使用立即执行函数
println("✅ 正确方式 2 - 立即执行函数:");
for (let i = 0; i < 5; i++) {
  (function(val) {
    go(function() {
      println("goroutine " + String(val));
    });
  })(i);  // 立即执行，创建新作用域
}

sleep(100);
println("");

// ✅ 正确示例 3: 使用辅助函数
println("✅ 正确方式 3 - 辅助函数:");
function startTask(id) {
  go(function() {
    println("goroutine " + String(id));
  });
}

for (let i = 0; i < 5; i++) {
  startTask(i);
}

sleep(100);
println("");

println("=== 测试完成 ===");
