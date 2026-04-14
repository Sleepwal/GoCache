# Git Version Manager

## 触发条件
当用户执行以下操作时触发：
- 提交新功能代码
- 修复bug
- 添加新特性
- 用户明确要求更新版本号或打tag

## 功能说明
此技能用于在git提交时自动管理版本号：
1. 根据提交类型（新功能、修复等）更新 README.md 和 QWEN.md 中的版本号
2. 同步更新 main.go 中的版本号示例
3. 创建对应的 git tag

## 执行流程

### 1. 分析提交类型
根据用户的提交信息确定版本更新类型：
- **主版本 (MAJOR)**: 不兼容的API变更
- **次版本 (MINOR)**: 向后兼容的新功能
- **补丁版本 (PATCH)**: 向后兼容的bug修复

### 2. 更新 README.md、QWEN.md 和 main.go
- 读取当前 README.md、QWEN.md 和 main.go 中的版本号
- 根据提交类型递增对应版本号
- 更新 README.md 中的版本信息
- 同步更新 QWEN.md 中的版本信息
- 同步更新 main.go 中的 `c.Set("version", "x.x.x", 0)` 示例版本号

### 3. 创建 Git Tag
- 使用新版本号创建 annotated tag
- Tag 消息包含本次变更的简要说明

## 使用方式

### 自动模式
用户正常提交代码后，技能会自动检测是否需要更新版本号：
```bash
# 用户提交代码
git add .
git commit -m "feat: 添加缓存过期功能"

# 技能自动执行
- 更新 README.md 版本号: v1.2.0 -> v1.3.0
- 更新 QWEN.md 版本号: v1.2.0 -> v1.3.0
- 更新 main.go 版本号: v1.2.0 -> v1.3.0
- 创建 tag: v1.3.0
```

### 手动模式
用户可以明确要求更新版本：
- "更新版本号并打tag"
- "提交代码并更新版本"
- "为这次提交创建tag"

## 版本号规则

遵循语义化版本 (Semantic Versioning):
- 格式: `v{MAJOR}.{MINOR}.{PATCH}`
- 示例: `v1.2.3`

### 版本号递增规则
- `feat`, `feature`, `新功能` -> MINOR版本+1
- `fix`, `bug`, `修复` -> PATCH版本+1
- `breaking`, `重大变更` -> MAJOR版本+1
- `docs`, `文档`, `chore`, `优化` -> 可选PATCH版本+1

## 实现细节

### 读取当前版本
```bash
# 从 README.md、QWEN.md 和 main.go 中提取当前版本号
# 支持格式:
# - ## 版本 v1.2.3 (README.md)
# - 当前版本: v0.7.0 (QWEN.md)
# - c.Set("version", "1.0.0", 0) (main.go)
```

### 更新 README.md、QWEN.md 和 main.go
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

1. 如果README.md中没有版本号，初始版本默认为 `v0.1.0`
2. 如果tag已存在，会提示冲突，不会强制覆盖
3. 更新前会显示变更预览，用户可以取消
4. 所有操作都会记录在git history中

## 示例

### 示例1: 新功能提交
用户: "提交缓存过期功能的代码"
技能:
1. 检测到新功能提交
2. 当前版本: v1.2.0
3. 更新为: v1.3.0
4. 更新 README.md
5. 更新 QWEN.md
6. 更新 main.go 中的版本号
7. 创建 tag v1.3.0
8. 提示用户: "✅ 版本已更新至 v1.3.0，tag 已创建，README.md、QWEN.md 和 main.go 已同步更新"

### 示例2: Bug修复
用户: "修复缓存并发问题"
技能:
1. 检测到 bug 修复
2. 当前版本: v1.3.0
3. 更新为: v1.3.1
4. 更新 README.md
5. 更新 QWEN.md
6. 更新 main.go 中的版本号
7. 创建 tag v1.3.1
8. 提示用户: "✅ 版本已更新至 v1.3.1，tag 已创建，README.md、QWEN.md 和 main.go 已同步更新"
