# @std/color

> 原生标准库接口单元 - 终端颜色输出。

## 模块路径

| 路径 | 说明 |
|------|------|
| `@std/color` | 原生模块路径 |

## 加载

```javascript
let c = require("@std/color");
```

## 接口

### 基础颜色

| 接口 | 说明 |
|------|------|
| `red(text)` | 红色文本 |
| `green(text)` | 绿色文本 |
| `yellow(text)` | 黄色文本 |
| `blue(text)` | 蓝色文本 |
| `magenta(text)` | 品红色文本 |
| `cyan(text)` | 青色文本 |
| `white(text)` | 白色文本 |
| `gray(text)` | 灰色文本 |
| `black(text)` | 黑色文本 |

### 背景色

| 接口 | 说明 |
|------|------|
| `bgRed(text)` | 红色背景 |
| `bgGreen(text)` | 绿色背景 |
| `bgYellow(text)` | 黄色背景 |
| `bgBlue(text)` | 蓝色背景 |
| `bgMagenta(text)` | 品红色背景 |
| `bgCyan(text)` | 青色背景 |
| `bgWhite(text)` | 白色背景 |

### 文本样式

| 接口 | 说明 |
|------|------|
| `bold(text)` | 粗体 |
| `dim(text)` | 暗淡 |
| `italic(text)` | 斜体 |
| `underline(text)` | 下划线 |
| `strikethrough(text)` | 删除线 |

### 自定义颜色

| 接口 | 说明 |
|------|------|
| `rgb(r, g, b)(text)` | RGB 颜色（需要 level >= 3） |
| `hex(color)(text)` | 十六进制颜色，如 "#FF6432"（需要 level >= 3） |

### 工具函数

| 接口 | 说明 |
|------|------|
| `strip(text)` | 移除 ANSI 颜色码 |
| `enabled` | 是否启用颜色（布尔值） |
| `level` | 颜色支持级别：0=无，1=基础，2=256色，3=真彩 |

## 链式调用

所有颜色、背景色和样式函数都支持链式调用：

```javascript
c.bold.red("粗体红色")
c.bgYellow.black("黄底黑字")
c.bold.underline.cyan("粗体下划线青色")
```

## 示例

```javascript
let c = require("@std/color");

// 基础颜色
console.log(c.red("错误"));
console.log(c.green("成功"));
console.log(c.yellow("警告"));

// 背景色
console.log(c.bgRed("紧急"));

// 样式
console.log(c.bold("粗体"));
console.log(c.underline("下划线"));

// 链式调用
console.log(c.bold.red("粗体红色"));

// 自定义颜色
console.log(c.rgb(255, 100, 50)("橙色"));
console.log(c.hex("#FF6432")("十六进制颜色"));

// 移除颜色码
let colored = c.red("红色文本");
let plain = c.strip(colored);
console.log(plain); // "红色文本"
```

## 维护来源

- `internal/stdlib/color.go` - 模块实现
- `examples/31-color.gs` - 使用示例
