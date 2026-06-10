// test-go.gs - 测试 go() 函数的基本功能

let counter = 0;

println("主线程开始");

// 测试 1: 基本 goroutine
go(function() {
  println("goroutine 1: 开始");
  for (let i = 0; i < 3; i++) {
    counter++;
    println("  goroutine 1: counter = " + counter);
  }
  println("goroutine 1: 完成");
});

// 测试 2: 带参数的 goroutine
go(function(name, count) {
  println("goroutine 2: 开始，name = " + name);
  for (let i = 0; i < count; i++) {
    counter++;
    println("  goroutine 2: counter = " + counter);
  }
  println("goroutine 2: 完成");
}, "test", 3);

// 测试 3: 多个并发 goroutine
for (let i = 0; i < 3; i++) {
  go(function(id) {
    println("goroutine " + id + ": 执行");
    counter++;
  }, i + 3);
}

println("主线程继续执行");

// 等待所有 goroutine 完成
sleep(1000);

println("最终 counter = " + counter);
println("测试完成");
