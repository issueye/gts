// test-go-final.gs - 最终测试

println("=== 测试 1: 基本 goroutine ===");
let x = 0;

go(function() {
  println("goroutine: x 设置为 42");
  x = 42;
});

sleep(100);
println("主线程: x = " + String(x));
println("");

println("=== 测试 2: 带参数的 goroutine ===");
go(function(a, b) {
  let result = a + b;
  println("goroutine: " + String(a) + " + " + String(b) + " = " + String(result));
}, 10, 20);

sleep(100);
println("");

println("=== 测试 3: 多个并发 goroutine (正确方式) ===");
let counter = 0;

// 方法 1: 将 i 作为参数传递（推荐）
function spawnTask(id) {
  go(function(taskId) {
    println("goroutine " + String(taskId) + " 执行");
    counter++;
  }, id);
}

for (let i = 0; i < 5; i++) {
  spawnTask(i);
}

sleep(200);
println("最终 counter = " + String(counter));
println("");

println("=== 测试完成 ===");
