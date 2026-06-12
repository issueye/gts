# @std/semver - 语义化版本

语义化版本（Semantic Versioning）解析和比较库。

## 导入

```javascript
let semver = require("@std/semver");
```

## API

### 解析与验证

#### `parse(version)`
解析版本号字符串。

```javascript
let v = semver.parse("1.2.3-alpha.1+build.123");
// {
//   major: 1,
//   minor: 2,
//   patch: 3,
//   prerelease: ["alpha", "1"],
//   build: ["build", "123"]
// }
```

#### `valid(version)`
验证版本号是否有效。

```javascript
semver.valid("1.2.3");        // true
semver.valid("1.2");          // false
semver.valid("v1.2.3");       // true (支持 v 前缀)
```

### 版本比较

#### `compare(v1, v2)`
比较两个版本。返回 -1、0 或 1。

```javascript
semver.compare("1.2.3", "1.3.0");  // -1
semver.compare("1.2.3", "1.2.3");  // 0
semver.compare("1.3.0", "1.2.3");  // 1
```

#### `gt(v1, v2)` / `gte(v1, v2)`
大于 / 大于等于。

```javascript
semver.gt("1.3.0", "1.2.3");   // true
semver.gte("1.2.3", "1.2.3");  // true
```

#### `lt(v1, v2)` / `lte(v1, v2)`
小于 / 小于等于。

```javascript
semver.lt("1.2.0", "1.2.3");   // true
semver.lte("1.2.3", "1.2.3");  // true
```

#### `eq(v1, v2)` / `neq(v1, v2)`
等于 / 不等于。

```javascript
semver.eq("1.2.3", "1.2.3");   // true
semver.neq("1.2.3", "1.3.0");  // true
```

### 版本递增

#### `inc(version, release)`
递增版本号。release 可以是: `major`、`minor`、`patch`、`prerelease`。

```javascript
semver.inc("1.2.3", "major");      // "2.0.0"
semver.inc("1.2.3", "minor");      // "1.3.0"
semver.inc("1.2.3", "patch");      // "1.2.4"
semver.inc("1.2.3", "prerelease"); // "1.2.4-0"
```

### 范围匹配

#### `satisfies(version, range)`
检查版本是否满足范围。

```javascript
semver.satisfies("1.2.5", "^1.2.0");  // true
semver.satisfies("1.3.0", "~1.2.0");  // false
semver.satisfies("1.2.5", ">=1.2.0 <2.0.0");  // true
```

支持的范围语法：
- `^1.2.0` - 兼容版本（允许 minor、patch 更新）
- `~1.2.0` - 近似版本（仅允许 patch 更新）
- `>=1.0.0 <2.0.0` - 范围表达式
- `1.2.x` 或 `1.2.*` - 通配符

## 示例

### 依赖版本检查

```javascript
let semver = require("@std/semver");

let currentVersion = "1.5.2";
let requiredRange = "^1.2.0";

if (semver.satisfies(currentVersion, requiredRange)) {
  console.log("版本兼容");
} else {
  console.log("版本不兼容");
}
```

### 版本排序

```javascript
let versions = ["1.3.0", "1.2.5", "2.0.0", "1.2.3"];

versions.sort((a, b) => semver.compare(a, b));
// ["1.2.3", "1.2.5", "1.3.0", "2.0.0"]
```

### 版本发布

```javascript
let current = "1.2.3";

console.log("下个补丁版本:", semver.inc(current, "patch"));
console.log("下个次版本:", semver.inc(current, "minor"));
console.log("下个主版本:", semver.inc(current, "major"));
```

## 注意事项

1. 版本号必须遵循 `major.minor.patch` 格式
2. 支持可选的 `v` 前缀（如 `v1.2.3`）
3. 预发布版本使用 `-` 分隔（如 `1.2.3-alpha.1`）
4. 构建元数据使用 `+` 分隔（如 `1.2.3+build.123`）
5. 范围匹配实现了基本的 semver 规范

## 相关

- [Semantic Versioning 2.0.0](https://semver.org/)
