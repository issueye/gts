# GTP 性能基线

> 日期：2026-06-05  
> 环境：Windows amd64，13th Gen Intel(R) Core(TM) i5-13420H  
> 命令：`go test ./internal/gtp -bench . -benchmem -run ^$`

## 当前测试内容

本基线测试的是 GTP JSON Lines 协议的核心开销：

- call frame JSON 编码
- call frame JSON 解码
- result frame JSON 编码
- JSONL 内存 loopback 编码+解码

当前实现位置：

- `internal/gtp/frame.go`
- `internal/gtp/jsonl.go`
- `internal/gtp/frame_benchmark_test.go`

## 初始结果

```text
BenchmarkEncodeCallFrame-12      	  106147	     10585 ns/op	    4974 B/op	      37 allocs/op
BenchmarkDecodeCallFrame-12      	   42898	     32895 ns/op	  12.89 MB/s	   15536 B/op	     186 allocs/op
BenchmarkEncodeResultFrame-12    	  164683	      7052 ns/op	    3276 B/op	      24 allocs/op
BenchmarkJSONLLoopback-12        	   31688	     42942 ns/op	   29685 B/op	     228 allocs/op
```

## 优化后结果

优化内容：

- `Value.MarshalJSON` 改为手写 JSON 输出，减少临时 wire struct 和反射路径。
- `Value.UnmarshalJSON` 对 number/string/boolean 做具体类型解码，避免标量统一落到 `interface{}`。
- JSONL decoder 从 `ReadBytes` 改为 `ReadSlice`，减少读行分配。
- JSONL encoder 继续使用优化后的 `EncodeFrame`，避免 `json.Encoder` 流式路径的额外开销。

命令：

```bash
go test ./internal/gtp -bench . -benchmem -run ^$
```

结果：

```text
BenchmarkEncodeCallFrame-12             	  246254	      5028 ns/op	    1817 B/op	      15 allocs/op
BenchmarkDecodeCallFrame-12             	   41155	     28412 ns/op	  14.92 MB/s	   14432 B/op	     172 allocs/op
BenchmarkEncodeResultFrame-12           	  390590	      2913 ns/op	    1250 B/op	      11 allocs/op
BenchmarkJSONLLoopback-12               	   36896	     37980 ns/op	   25093 B/op	     191 allocs/op
BenchmarkJSONLLoopbackReuseBuffer-12    	   33639	     39881 ns/op	   16390 B/op	     187 allocs/op
```

## 对比

| Benchmark | 初始 ns/op | 优化后 ns/op | 初始 B/op | 优化后 B/op | 初始 allocs/op | 优化后 allocs/op |
|-----------|------------|--------------|-----------|-------------|----------------|------------------|
| EncodeCallFrame | 10585 | 5028 | 4974 | 1817 | 37 | 15 |
| DecodeCallFrame | 32895 | 28412 | 15536 | 14432 | 186 | 172 |
| EncodeResultFrame | 7052 | 2913 | 3276 | 1250 | 24 | 11 |
| JSONLLoopback | 42942 | 37980 | 29685 | 25093 | 228 | 191 |

## 结论

- call frame 编码从约 10.6 微秒降到约 5.0 微秒，分配从 37 次降到 15 次。
- result frame 编码从约 7.1 微秒降到约 2.9 微秒，分配从 24 次降到 11 次。
- call frame 解码从约 32.9 微秒降到约 28.4 微秒，仍是主要开销。
- 内存 JSONL loopback 从约 42.9 微秒降到约 38.0 微秒，不包含真实进程、管道或网络调度成本。

这说明第一版 JSON Lines 方案适合工具调用、宿主能力调用、长任务调度和中低频 RPC；如果要支撑高频小包调用，需要继续降低解码分配。

## 后续优化方向

1. 为 `Frame` 解码做按 `type` 分派的 fast path，减少通用宽结构体字段处理。
2. 为 `Value` object/array 解码做 token-based fast path，继续降低 map/slice 分配。
3. 复用 encoder/decoder buffer，避免 JSONL loopback 每次创建 codec。
4. 增加 pipe loopback benchmark，测真实 stdin/stdout 或 named pipe 成本。
5. 增加大 payload benchmark，覆盖 bytes/base64、数组和对象深度。
