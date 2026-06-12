# `@std/watch` - 文件监听

> 提供文件变化监听功能。

## 基础用法

```javascript
let watch = require("@std/watch");

watch.file("config.json", () => {
  console.log("Config changed!");
});
```

## API

### watch.file(path, callback, options?)

监听文件变化。

**参数**：
- `path` - 文件路径
- `callback` - 变化时的回调
- `options.interval` - 检查间隔（毫秒，默认 1000）

**示例**：
```javascript
watch.file("data.json", () => {
  let data = JSON.parse(fs.readFileSync("data.json"));
  console.log("Reloaded:", data);
}, { interval: 500 });
```
