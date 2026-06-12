# `@std/regexp` - 正则表达式增强

> 提供正则表达式的增强功能。

## 基础用法

```javascript
let regexp = require("@std/regexp");

// 转义特殊字符
let escaped = regexp.escape("hello.world"); // "hello\\.world"

// 匹配所有
let matches = regexp.matchAll(/\d+/g, "123 456 789");

// 正则分割
let parts = regexp.split(/\s+/, "a b  c");
```

## API

### regexp.escape(str)

转义正则表达式特殊字符。

**参数**：
- `str` - 要转义的字符串

**返回**：转义后的字符串

**示例**：
```javascript
regexp.escape("hello.world")  // "hello\\.world"
regexp.escape("$100")         // "\\$100"
```

### regexp.matchAll(pattern, str)

查找所有匹配项（包括捕获组）。

**参数**：
- `pattern` - 正则表达式模式
- `str` - 要搜索的字符串

**返回**：数组，每项是一个匹配结果数组（包含捕获组）

**示例**：
```javascript
let matches = regexp.matchAll(/(\d+)-(\d+)/, "10-20, 30-40");
// [[" 10-20", "10", "20"], ["30-40", "30", "40"]]
```

### regexp.split(pattern, str, limit?)

使用正则表达式分割字符串。

**参数**：
- `pattern` - 正则表达式模式
- `str` - 要分割的字符串
- `limit` - 可选，最大分割数

**返回**：分割后的字符串数组

**示例**：
```javascript
regexp.split(/\s+/, "a  b   c")      // ["a", "b", "c"]
regexp.split(/,\s*/, "a, b,  c")     // ["a", "b", "c"]
regexp.split(/-/, "a-b-c", 2)        // ["a", "b-c"]
```
