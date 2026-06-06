function main() {
  const bots = require("@plugin/im-bot");

  bots.configure({
    name: "qq-local",
    platform: "onebot",
    baseUrl: "http://127.0.0.1:3000"
  });

  let adapters = bots.list();
  println("IM bot adapters: " + String(adapters.length));

  // 配好真实平台后再打开：
  // bots.send({
  //   adapter: "qq-local",
  //   toType: "group",
  //   to: "123456",
  //   text: "GTS 脚本发来的消息"
  // });
}
