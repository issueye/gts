# `@std/diff` - 文本差异比较

> 提供文本差异比较功能。

## 基础用法

```javascript
let diff = require("@std/diff");

// 行级差异
let lineDiff = diff.lines(text1, text2);

// 字符级差异
let charDiff = diff.chars(str1, str2);
```

## API

### diff.lines(text1, text2)

比较两个文本的行级差异。

**参数**：
- `text1` - 原始文本
- `text2` - 新文本

**返回**：数组，每项包含：
- `type` - 差异类型：`"equal"`, `"added"`, `"removed"`
- `value` - 行内容

### diff.chars(str1, str2)

比较两个字符串的字符级差异。

**参数**：
- `str1` - 原始字符串
- `str2` - 新字符串

**返回**：数组，每项包含：
- `type` - 差异类型
- `value` - 字符

## 示例

```javascript
let diff = require("@std/diff");

let result = diff.lines(
  "line1\nline2\nline3",
  "line1\nmodified\nline3"
);

// 输出差异
for (let item of result) {
  if (item.type === "removed") {
    console.log("- " + item.value);
  } else if (item.type === "added") {
    console.log("+ " + item.value);
  } else {
    console.log("  " + item.value);
  }
}
```
