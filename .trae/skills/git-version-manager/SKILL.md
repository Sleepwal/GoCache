---
name: "git-version-manager"
description: "在 Git 提交时自动管理版本号，更新 README.md、AGENT.md 和 main.go 中的版本号，并创建对应的 Git Tag。当用户提交新功能、修复 Bug 或明确要求更新版本号/打 Tag 时触发此技能。"
---

# Git Version Manager

此技能用于在 Git 提交时自动管理版本号。

## 提交信息规范

### 格式要求
- **提交信息主体**：使用中文描述
- **类型前缀**：使用英文关键词（feat、fix、perf、refactor、docs、chore、test）

### 推荐格式
```
<类型前缀>: <中文描述>
```

示例：
```
feat: 新增缓存过期功能
fix: 修复并发竞态条件
perf: 优化内存分配策略
docs: 更新API文档
```

## 触发条件

当用户执行以下操作时触发：
- 提交新功能代码
- 修复 Bug
- 添加新特性
- 用户明确要求更新版本号或打 Tag

## 执行流程

### 1. 分析提交类型

根据用户的提交信息确定版本更新类型：
- **主版本 (MAJOR)**：不兼容的 API 变更
- **次版本 (MINOR)**：向后兼容的新功能
- **补丁版本 (PATCH)**：向后兼容的 Bug 修复

### 2. 更新 README.md、AGENT.md 和 main.go

- 读取当前 README.md、AGENT.md 和 main.go 中的版本号
- 根据提交类型递增对应版本号
- 更新 README.md 中的版本信息
- 同步更新 AGENT.md 中的版本信息
- 同步更新 main.go 中的 `c.Set("version", "x.x.x", 0)` 示例版本号

### 3. 创建 Git Tag

- 使用新版本号创建 annotated tag
- Tag 消息包含本次变更的简要说明

## 版本号规则

遵循语义化版本 (Semantic Versioning)：
- 格式：`v{MAJOR}.{MINOR}.{PATCH}`
- 示例：`v1.2.3`

### 版本号递增规则

- `feat`, `feature` -> MINOR 版本 +1（新功能）
- `fix`, `bug` -> PATCH 版本 +1（Bug 修复）
- `perf`, `优化`, `性能` -> PATCH 版本 +1（性能优化）
- `refactor`, `重构` -> PATCH 版本 +1（代码重构）
- `breaking`, `重大变更` -> MAJOR 版本 +1（破坏性变更）
- `docs`, `文档`, `style`, `chore`, `test` -> 不更新版本

## 使用方式

### 自动模式

用户正常提交代码后，技能会自动检测是否需要更新版本号：

```bash
# 用户提交代码（中文描述 + 英文前缀）
git add .
git commit -m "feat: 新增缓存过期功能"

# 技能自动执行
# - 更新 README.md 版本号: v1.2.0 -> v1.3.0
# - 更新 AGENT.md 版本号: v1.2.0 -> v1.3.0
# - 更新 main.go 版本号: v1.2.0 -> v1.3.0
# - 创建 tag: v1.3.0
```

### 手动模式

用户可以明确要求更新版本：
- "更新版本号并打 Tag"
- "提交代码并更新版本"
- "为这次提交创建 Tag"

## 实现细节

### 读取当前版本

```bash
# 从 README.md、AGENT.md 和 main.go 中提取当前版本号
# 支持格式:
# - 当前版本: v1.2.3 (README.md)
# - **当前版本**: v0.7.0 (AGENT.md)
# - c.Set("version", "1.0.0", 0) (main.go)
```

### 更新版本号

```bash
# 使用正则表达式替换版本号
# 保持原有格式不变
# 同步更新三个文件中的版本信息
# main.go: 更新 c.Set("version", "x.x.x", 0) 中的版本号
```

### 创建 Git Tag

```bash
git tag -a v{新版本号} -m "Release v{新版本号}"
```

## 注意事项

1. 如果 README.md 中没有版本号，初始版本默认为 `v0.1.0`
2. 如果 Tag 已存在，会提示冲突，不会强制覆盖
3. 更新前会显示变更预览，用户可以取消
4. 所有操作都会记录在 Git history 中

## 示例

### 示例 1: feat - 新功能

```bash
git commit -m "feat: 新增缓存过期功能"
```

技能自动执行：
1. 检测到 `feat`，更新 MINOR 版本
2. 更新 README.md、AGENT.md、main.go
3. 创建 tag v1.x.0

### 示例 2: fix - Bug 修复

```bash
git commit -m "fix: 修复缓存并发问题"
```

技能自动执行：
1. 检测到 `fix`，更新 PATCH 版本
2. 更新 README.md、AGENT.md、main.go
3. 创建 tag v1.x.x

### 示例 3: docs - 文档更新

```bash
git commit -m "docs: 更新API文档"
```

`docs` 类型不触发版本更新，仅提交文档。

## 详细示例参考

更多详细的提交示例和模板请查看：[reference/commit-examples.md](reference/commit-examples.md)

包含：
- 9 种提交类型的完整示例
- 多模块同时更新的提交示例
- 版本更新对照表
- feat/fix/perf 类型的提交模板
