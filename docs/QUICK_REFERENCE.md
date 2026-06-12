# GTS 标准库扩展 - 快速参考

## 📂 文件导航

### 核心文档
```
gts/docs/
├── STDLIB_EXPANSION_README.md          # 📖 项目总览和使用指南
├── native-stdlib-expansion-plan.md     # 📋 完整的 API 规范（20个模块）
├── native-stdlib-development-plan.md   # 📅 详细的开发计划
├── native-stdlib-progress.md           # 📊 实施进度跟踪
└── native-stdlib-summary.md            # 📝 工作总结报告
```

### 模块文档
```
gts/docs/wiki/native-stdlib/
└── test.md                             # ✅ 测试框架完整文档
```

### 代码实现
```
gts/internal/stdlib/
└── test.go                             # ✅ 测试框架实现 (22KB)
```

### 示例代码
```
gts/examples/
└── 25-test-framework.gs                # ✅ 测试框架示例
```

---

## 🚀 快速开始

### 1. 查看总体规划
```bash
cat docs/STDLIB_EXPANSION_README.md
```

### 2. 了解 API 设计
```bash
cat docs/native-stdlib-expansion-plan.md
```

### 3. 运行测试示例
```bash
cd gts
./gs.exe examples/25-test-framework.gs
```

---

## 📚 使用已完成的模块

### @std/test - 测试框架

```javascript
let test = require("@std/test");

// 基础测试
test("should add numbers", () => {
  test.expect(1 + 1).toBe(2);
});

// 测试套件
test.describe("Math", () => {
  test.it("should multiply", () => {
    test.expect(3 * 4).toBe(12);
  });
});

// 运行测试
test.run();
```

**完整文档**: `docs/wiki/native-stdlib/test.md`

---

## 📊 当前进度

- **总进度**: 5% (1/20 模块完成)
- **P0 进度**: 25% (1/4 模块完成)
- **文档**: 30%
- **编译**: ✅ 通过

---

## 🎯 下一步

### 本周目标
1. ⏳ 完善 @std/test (集成 evaluator)
2. ⏳ 实现 @std/env
3. ⏳ 实现 @std/json
4. ⏳ 实现 @std/validation

### 查看详细计划
```bash
cat docs/native-stdlib-development-plan.md
cat docs/native-stdlib-progress.md
```

---

## 🔗 关键链接

| 文档 | 说明 | 大小 |
|------|------|------|
| [README](STDLIB_EXPANSION_README.md) | 项目总览 | 6.6K |
| [API规范](native-stdlib-expansion-plan.md) | 20个模块设计 | 18K |
| [开发计划](native-stdlib-development-plan.md) | 实施细节 | 17K |
| [进度报告](native-stdlib-progress.md) | 当前状态 | 7.5K |
| [工作总结](native-stdlib-summary.md) | 成果展示 | 7.3K |
| [test.md](wiki/native-stdlib/test.md) | 测试框架 | 8.6K |

---

**更新时间**: 2026-06-12  
**状态**: ✅ 第一个模块完成并编译通过
