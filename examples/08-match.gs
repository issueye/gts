// ============================================================
// 08-match.gs —— 第八步：模式匹配
// ============================================================
// match 表达式替代 switch，无 fall-through，比 if/else 更清晰

// --- 基础值匹配 ---
console.log("=== 基础值匹配 ===");
function getDayName(day: number): string {
  return match day {
    1 => "星期一",
    2 => "星期二",
    3 => "星期三",
    4 => "星期四",
    5 => "星期五",
    6 => "星期六",
    7 => "星期日",
    _ => "无效日期",
  };
}
for (let i = 1; i <= 8; i = i + 1) {
  console.log(`  day ${i}:`, getDayName(i));
}

// --- OR 模式（多值匹配同一分支） ---
console.log("");
console.log("=== OR 模式 ===");
function isWeekend(day: number): boolean {
  return match day {
    6 | 7 => true,
    _ => false,
  };
}
console.log("isWeekend(5):", isWeekend(5));  // false
console.log("isWeekend(7):", isWeekend(7));  // true

// --- 范围模式 ---
console.log("");
console.log("=== 范围模式 ===");
function grade(score: number): string {
  return match score {
    0..60     => "不及格",     // 0 <= score < 60
    60..70    => "及格",
    70..80    => "中等",
    80..90    => "良好",
    90..=100  => "优秀",       // 90 <= score <= 100（闭区间）
    _         => "无效分数",
  };
}
let scores = [55, 72, 85, 95, 100];
for (let s of scores) {
  console.log(`  ${s}分:`, grade(s));
}

// --- 绑定 + 守卫 ---
console.log("");
console.log("=== 绑定 + 守卫 ===");
function describe(n: number): string {
  return match n {
    0          => "零",
    n if n < 0 => `负数 ${n}`,
    n if n > 0 => `正数 ${n}`,
    _          => "NaN",
  };
}
console.log(describe(0));    // 零
console.log(describe(-5));   // 负数 -5
console.log(describe(42));   // 正数 42

// --- 语句模式 match（执行代码块） ---
console.log("");
console.log("=== 语句模式 ===");
let state = "idle";

function handleEvent(event: string): void {
  match event {
    "start" => { state = "running"; console.log("  启动"); },
    "pause" => { state = "paused"; console.log("  暂停"); },
    "stop"  => { state = "idle";   console.log("  停止"); },
    "status"=> console.log(`  当前状态: ${state}`),
    _       => console.log(`  未知事件: ${event}`),
  };
}

handleEvent("start");
handleEvent("status");
handleEvent("pause");
handleEvent("status");
handleEvent("stop");

// --- 嵌套模式匹配 ---
console.log("");
console.log("=== 复杂匹配函数 ===");
function httpStatusLabel(code: number): string {
  return match code {
    200 (val) => "OK",
    301 (val) => "Moved Permanently",
    403 (val) => "Forbidden",
    404 (val) => "Not Found",
    500..599 (val) => "Server Error",
    _ => `Unknown (${code})`,
  };
}
console.log("200 =>", httpStatusLabel(200));
console.log("502 =>", httpStatusLabel(502));
