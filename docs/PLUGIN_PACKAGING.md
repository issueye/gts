# GTS 插件打包策略

## 概述

GTS 使用 GTP（Go Transport Protocol）协议支持插件系统。插件是独立的二进制程序，通过进程间通信与主程序交互。

## 打包策略

### ✅ 插件不打包进主程序

插件二进制文件**不会**打包到主程序中，而是作为独立文件部署。

### 自动复制插件

使用 `gs dist` 命令打包时，会自动：

1. 读取 `config.toml` 中的插件配置
2. 查找插件二进制文件
3. 复制到 `dist/plugins/` 目录

## 部署结构

```
dist/
  ├── myapp.exe          # 主程序
  └── plugins/           # 插件目录
      ├── plugin1.exe
      ├── plugin2.exe
      └── ...
```

## 配置示例

**config.toml**:
```toml
[plugins.myPlugin]
command = "plugins/myPlugin.exe"
auto_start = true
capabilities = ["call", "event"]
modules = ["myModule"]
```

## 打包命令

```bash
# 打包项目（自动复制插件）
gs dist

# 打包到指定目录
gs dist . ./dist/myapp.exe
```

## 插件路径解析

打包后，插件路径会自动调整：

- **开发时**: `./plugins/myPlugin.exe`
- **打包后**: `./plugins/myPlugin.exe` (相对于可执行文件)

## 优势

1. **独立更新**: 可以单独更新插件，无需重新打包主程序
2. **减小体积**: 主程序不包含插件代码
3. **按需加载**: 只加载需要的插件
4. **灵活部署**: 可以选择性部署插件

## 注意事项

1. **插件依赖**: 确保插件二进制在目标平台可执行
2. **相对路径**: 建议使用相对路径配置插件
3. **权限**: Linux/Mac 需要确保插件有执行权限（自动设置）

## 手动部署

如果不使用 `gs dist`，手动部署时需要：

```bash
# 1. 复制主程序
cp gs.exe dist/

# 2. 创建插件目录
mkdir dist/plugins

# 3. 复制插件
cp plugins/*.exe dist/plugins/
```

## 开发模式

开发时，插件直接从源码目录加载，无需特殊配置。

## 示例项目结构

```
myproject/
  ├── config.toml       # 插件配置
  ├── main.gs           # 主脚本
  └── plugins/          # 插件目录
      ├── myPlugin.exe
      └── ...
```

打包后：

```
dist/
  ├── myproject.exe     # 包含脚本的可执行文件
  └── plugins/          # 自动复制的插件
      ├── myPlugin.exe
      └── ...
```

## 总结

✅ **插件不打包进主程序**  
✅ **自动复制到 dist/plugins/**  
✅ **独立部署和更新**  
✅ **按需加载**
