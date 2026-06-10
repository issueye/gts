// test-closure-fix.gs - 测试闭包捕获修复

println("=== 测试 1: for 循环 + go() ===");

for (let i = 0; i < 5; i++) {
  go(function() {
    println("goroutine 捕获 i = " + String(i));
  });
}

sleep(200);

println("");
println("=== 测试 2: for 循环 + 数组 ===");

let funcs = [];
for (let i = 0; i < 5; i++) {
  funcs.push(function() {
    return i;
  });
}

sleep(100);

for (let j = 0; j < funcs.length; j++) {
  println("funcs[" + String(j) + "]() = " + String(funcs[j]()));
}

println("");
println("=== 测试 3: for 循环 + WaitGroup ===");

let wg = makeWaitGroup();

for (let i = 0; i < 3; i++) {
  wg.add(1);
  go(function() {
    println("Worker i = " + String(i));
    wg.done();
  });
}

wg.wait();

println("");
println("=== 测试完成 ===");
