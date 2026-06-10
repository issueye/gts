// test-channel.gs - 测试 Channel

println("=== 测试 1: 基本 Channel 通信 ===");

let ch = makeChannel(0); // 无缓冲

go(function() {
  println("发送者: 准备发送");
  ch.send(42);
  println("发送者: 已发送 42");
});

sleep(100);
let val = ch.recv();
println("接收者: 收到 " + String(val));
ch.close();

println("");
println("=== 测试 2: 缓冲 Channel ===");

let ch2 = makeChannel(3); // 缓冲 3

ch2.send(1);
ch2.send(2);
ch2.send(3);
println("发送了 3 个值到缓冲 channel");

println("收到: " + String(ch2.recv()));
println("收到: " + String(ch2.recv()));
println("收到: " + String(ch2.recv()));

ch2.close();

println("");
println("=== 测试 3: WaitGroup ===");

let wg = makeWaitGroup();
let counter = 0;

for (let i = 0; i < 3; i++) {
  wg.add(1);
  go(function(id) {
    println("Worker " + String(id) + " 开始");
    counter++;
    sleep(50);
    println("Worker " + String(id) + " 完成");
    wg.done();
  }, i);
}

println("等待所有 worker 完成...");
wg.wait();
println("所有 worker 已完成，counter = " + String(counter));

println("");
println("=== 测试完成 ===");
