# `@std/retry` - 重试逻辑

> 提供函数重试执行功能。

## 基础用法

```javascript
let retry = require("@std/retry");

// 基础重试
await retry.run(() => fetchData(), { times: 3 });

// 带延迟
await retry.run(() => api(), { 
  times: 5, 
  delay: 1000 
});
```

## API

### retry.run(fn, options)

执行函数，失败时自动重试。

**参数**：
- `fn` - 要执行的函数
- `options.times` - 最大重试次数（默认 3）
- `options.delay` - 重试延迟（毫秒，默认 0）
- `options.backoff` - 指数退避倍数（默认 1）

**返回**：函数执行结果

### retry.exponential(fn, options)

使用指数退避策略重试。

**示例**：
```javascript
await retry.exponential(() => api(), {
  times: 5,
  initialDelay: 1000,
  maxDelay: 30000
});
```
