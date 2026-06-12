# `@std/rate-limit` - 速率限制

> 提供令牌桶算法的速率限制功能。

## 基础用法

```javascript
let rateLimit = require("@std/rate-limit");

// 创建限流器（每秒 10 个请求）
let limiter = rateLimit.create({ rate: 10, capacity: 10 });

// 尝试获取令牌
if (limiter.tryAcquire()) {
  // 执行操作
}

// 阻塞等待令牌
limiter.acquire();
doWork();
```

## API

### rateLimit.create(options)

创建速率限制器。

**参数**：
- `options.rate` - 每秒产生的令牌数（默认 10）
- `options.capacity` - 令牌桶容量（默认 10）

**返回**：限流器实例

### limiter.tryAcquire()

尝试获取令牌（非阻塞）。

**返回**：boolean - 是否成功获取

### limiter.acquire()

获取令牌（阻塞等待）。

### limiter.remaining()

获取剩余令牌数。

**返回**：number - 剩余令牌数
