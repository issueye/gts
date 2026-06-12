# GTS 原生标准库扩展项目

> 为 GTS 脚本语言添加 20 个新的原生标准库模块，提升工程化能力和开发体验。

---

## 📖 项目概览

本项目旨在扩展 GTS 的原生标准库，补充测试、验证、工具等核心功能，使 GTS 成为一个功能完善的脚本语言。

### 当前状态

- **总计划**：20 个新模块
- **已完成**：1 个模块（5%）
- **进行中**：P0 级别 4 个模块
- **时间**：2026-06-12 启动

---

## 🎯 模块清单

### ✅ 已完成（1/20）

| 模块 | 优先级 | 状态 | 说明 |
|------|--------|------|------|
| `@std/test` | 🔴 P0 | ✅ 完成 | 测试框架 |

### ⏳ 开发中（3/20）

| 模块 | 优先级 | 状态 | 说明 |
|------|--------|------|------|
| `@std/env` | 🔴 P0 | ⏳ 待开发 | 环境变量管理 |
| `@std/json` | 🔴 P0 | ⏳ 待开发 | JSON 增强（JSON5/Schema/Patch） |
| `@std/validation` | 🔴 P0 | ⏳ 待开发 | 数据验证 |

### 📋 计划中（16/20）

**P1 级别**（5个）：
- `@std/collections` - 集合操作
- `@std/random` - 随机数增强
- `@std/color` - 终端颜色
- `@std/semver` - 版本管理
- `@std/cache` - 内存缓存

**P2 级别**（11个）：
- `@std/diff` - 文本差异
- `@std/glob` - 文件匹配
- `@std/watch` - 文件监听
- `@std/retry` - 重试逻辑
- `@std/rate-limit` - 速率限制
- `@std/jwt` - JWT 令牌
- `@std/regexp` - 正则增强
- `@std/pdf` - PDF 处理
- `@std/image` - 图像处理
- `@std/prometheus` - 指标采集
- `@std/compression` - 更多压缩格式

---

## 🚀 快速开始

### 使用已完成的模块

```javascript
// 测试框架
let test = require("@std/test");

test.describe("Example", () => {
  test.it("should work", () => {
    test.expect(1 + 1).toBe(2);
  });
});

test.run();
```

### 运行示例

```bash
# 编译 GTS
cd gts
go build -o gs.exe ./cmd/gs

# 运行测试框架示例
./gs.exe examples/25-test-framework.gs
```

---

## 📚 文档结构

```
gts/docs/
├── native-stdlib-expansion-plan.md    # 完整的 API 规范设计
├── native-stdlib-development-plan.md  # 详细的开发计划
├── native-stdlib-progress.md          # 实施进度跟踪
├── native-stdlib-summary.md           # 工作总结报告
└── wiki/native-stdlib/
    ├── test.md                         # 测试框架文档
    ├── env.md                          # 环境变量文档（待创建）
    └── ...                             # 其他模块文档
```

### 核心文档

1. **[扩展规划](native-stdlib-expansion-plan.md)**
   - 20 个模块的完整 API 设计
   - 使用场景和示例代码
   - 优先级分类（P0/P1/P2）

2. **[开发计划](native-stdlib-development-plan.md)**
   - 详细的实施步骤
   - 时间安排和资源分配
   - 技术细节和依赖管理
   - 风险评估和应对策略

3. **[实施进度](native-stdlib-progress.md)**
   - 实时更新的进度报告
   - 已完成和待完成的任务
   - 遇到的问题和解决方案
   - 质量指标追踪

4. **[工作总结](native-stdlib-summary.md)**
   - 阶段性成果总结
   - 经验教训和改进建议
   - 项目影响评估

---

## 💻 开发指南

### 添加新模块的步骤

1. **创建知识库文档**
   ```bash
   # 在 docs/wiki/native-stdlib/ 创建模块文档
   touch docs/wiki/native-stdlib/module-name.md
   ```

2. **实现 Go 侧代码**
   ```bash
   # 在 internal/stdlib/ 创建模块实现
   touch internal/stdlib/module_name.go
   ```

3. **注册模块**
   ```go
   // 在模块文件中注册
   func init() {
       module.RegisterNative("@std/module-name", func(env *object.Environment) (object.Object, error) {
           exports := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
           initModuleNameModule(exports)
           return exports, nil
       })
   }
   ```

4. **编写测试**
   ```bash
   touch internal/stdlib/module_name_test.go
   ```

5. **创建示例**
   ```bash
   touch examples/XX-module-name.gs
   ```

6. **更新索引**
   - 更新 `docs/wiki/native-stdlib.md`
   - 更新 `docs/native-stdlib-progress.md`

### 代码规范

- 遵循 Go 标准代码风格
- 每个导出函数都有注释
- 使用有意义的变量名
- 完善的错误处理
- 单元测试覆盖率 > 80%

---

## 🧪 测试

### 运行单元测试

```bash
cd gts
go test ./internal/stdlib/...
```

### 运行示例验证

```bash
# 测试框架示例
./gs.exe examples/25-test-framework.gs

# 其他示例
./gs.exe examples/26-env-management.gs    # 待实现
./gs.exe examples/27-json-enhanced.gs     # 待实现
./gs.exe examples/28-validation.gs        # 待实现
```

---

## 📊 进度追踪

### 总体进度

```
████░░░░░░░░░░░░░░░░ 5% (1/20 完成)
```

### P0 模块进度

```
█████░░░░░░░░░░░ 25% (1/4 完成)
```

### 详细进度

| 阶段 | 完成度 | 说明 |
|------|--------|------|
| 规范设计 | 100% | 20 个模块 API 设计完成 |
| P0 实现 | 25% | 1/4 模块完成 |
| P1 实现 | 0% | 0/5 模块完成 |
| P2 实现 | 0% | 0/11 模块完成 |
| 文档编写 | 30% | 核心文档完成 |
| 测试覆盖 | 5% | 待补充 |

---

## 🎯 里程碑

### ✅ 第一阶段（已完成 - 2026-06-12）

- ✅ 完成 20 个模块的 API 规范设计
- ✅ 制定详细的开发计划
- ✅ 实现并编译通过 `@std/test` 模块
- ✅ 建立标准化的开发流程

### 🔄 第二阶段（进行中 - 目标 2026-06-19）

- ⏳ 完成 `@std/env` 模块
- ⏳ 完成 `@std/json` 模块
- ⏳ 完成 `@std/validation` 模块
- ⏳ 补充单元测试和集成测试

### 📅 第三阶段（计划中 - 目标 2026-07-12）

- ⏳ 完成 P1 级别 5 个模块
- ⏳ 所有模块都有完整文档和示例
- ⏳ 测试覆盖率达到 80%+

---

## 🤝 贡献指南

### 如何贡献

1. **选择模块**：从待开发列表选择一个模块
2. **查看规范**：阅读 `native-stdlib-expansion-plan.md` 中的设计
3. **实现功能**：按照开发指南实现模块
4. **编写测试**：确保测试覆盖率 > 80%
5. **更新文档**：完善知识库文档和示例
6. **提交审查**：提交代码审查

### 代码审查标准

- ✅ 编译通过
- ✅ 测试通过
- ✅ 文档完整
- ✅ 示例可运行
- ✅ 代码规范

---

## 📞 联系方式

- **项目地址**：gts/docs/native-stdlib-*
- **问题反馈**：通过 Issues 提交
- **讨论交流**：开发者社区

---

## 📜 许可证

遵循 GTS 项目的许可证。

---

## 🙏 致谢

感谢所有为 GTS 标准库扩展做出贡献的开发者！

---

**最后更新**：2026-06-12  
**下次更新**：完成 @std/env 模块后
