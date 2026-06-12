# `@std/glob` - 文件模式匹配

> 提供 glob 模式的文件匹配功能。

## 基础用法

```javascript
let glob = require("@std/glob");

// 匹配单个路径
glob.match("*.js", "test.js");  // true

// 查找文件
let files = glob.find("src/**/*.js");
```

## API

### glob.match(pattern, path)

检查路径是否匹配模式。

**参数**：
- `pattern` - glob 模式
- `path` - 要检查的路径

**返回**：boolean

### glob.find(pattern)

查找匹配模式的所有文件。

**参数**：
- `pattern` - glob 模式

**返回**：string[] - 匹配的文件路径数组

## 模式语法

- `*` - 匹配任意字符（不包括 /）
- `**` - 匹配任意路径（包括 /）
- `?` - 匹配单个字符
- `[abc]` - 匹配字符集
- `{a,b}` - 匹配多个模式

## 示例

```javascript
// 匹配 .js 文件
glob.find("*.js")

// 匹配所有子目录的 .ts 文件
glob.find("**/*.ts")

// 匹配特定目录
glob.find("src/**/*.{js,ts}")
```
