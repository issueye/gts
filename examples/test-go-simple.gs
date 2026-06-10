// test-go-simple.gs - 简单测试

println("=== 测试 1: 基本 goroutine ===");
let x = 0;

go(function() {
  println("goroutine: 开始");
  x = 42;
  println("goroutine: x = " + String(x));
});

sleep(100);
println("主线程: x = " + String(x));

println("");
println("=== 测试 2: 带参数的 goroutine ===");
go(function(a, b) {
  println("goroutine: a = " + String(a) + ", b = " + String(b));
  println("goroutine: a + b = " + String(a + b));
}, 10, 20);

sleep(100);

println("");
println("=== 测试 3: 多个并发 goroutine ===");
let counter = 0;
for (let i = 0; i < 5; i++) {
  go(function(id) {
    println("goroutine " + String(id) + " 执行");
    counter++;
  }, i);
}

sleep(200);
println("最终 counter = " + String(counter));

println("");
println("=== 测试完成 ===");
