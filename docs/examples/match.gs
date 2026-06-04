// match.gs —— 模式匹配（替代 switch）

// 1) 表达式模式：HTTP 状态码
function httpLabel(code: number): string {
  return match code {
    200 (val) => "OK",
    201 (val) => "Created",
    301 (val) => "Moved Permanently",
    302 (val) => "Found",
    304 (val) => "Not Modified",
    400 (val) => "Bad Request",
    401 (val) => "Unauthorized",
    403 (val) => "Forbidden",
    404 (val) => "Not Found",
    500..599 (val) => "Server Error",
    _ => `Unknown (${code})`,
  };
}

console.log(httpLabel(200));   // OK
console.log(httpLabel(301));   // Moved Permanently
console.log(httpLabel(502));   // Server Error
console.log(httpLabel(999));   // Unknown (999)

// 2) OR 模式
function season(month: number): string {
  return match month {
    12 | 1 | 2 => "Winter",
    3 | 4 | 5  => "Spring",
    6 | 7 | 8  => "Summer",
    9 | 10 | 11 => "Autumn",
    _ => "Invalid",
  };
}

console.log(season(1));    // Winter
console.log(season(7));    // Summer

// 3) 绑定 + 守卫
function describe(n: number): string {
  return match n {
    0          => "zero",
    n if n < 0 => `negative ${n}`,
    n if n > 0 => `positive ${n}`,
    _          => "NaN",
  };
}

console.log(describe(0));    // zero
console.log(describe(-5));   // negative -5
console.log(describe(42));   // positive 42

// 4) 范围模式（闭区间与半开区间）
function grade(score: number): string {
  return match score {
    0..60     => "F",
    60..70    => "D",
    70..80    => "C",
    80..90    => "B",
    90..=100  => "A",
    _         => "Invalid",
  };
}

console.log(grade(55));    // F
console.log(grade(75));    // C
console.log(grade(95));    // A
console.log(grade(100));   // A (含 100)

// 5) 作为语句使用（块体）
let count: number = 0;

function handle(cmd: string): void {
  match cmd {
    "inc" => { count = count + 1; },
    "dec" => { count = count - 1; },
    "reset" => { count = 0; },
    "status" => console.log(`count = ${count}`),
    _ => console.log(`unknown command: ${cmd}`),
  };
}

handle("inc");
handle("inc");
handle("status");   // count = 2
handle("dec");
handle("status");   // count = 1
handle("reset");
handle("status");   // count = 0
handle("flip");     // unknown command: flip
