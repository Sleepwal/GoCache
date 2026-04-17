# Git 版本管理示例参考

本文档提供详细的提交示例，遵循语义化版本规范。

## 提交格式规范

```
<类型前缀>: <中文描述>

<可选的详细说明（使用列表）>
```

## 示例列表

### 1. feat - 新功能提交

```bash
git commit -m "feat: 添加缓存命名空间"

- NamespaceCache 包装器实现隔离分区
- MultiNamespaceCache 支持多命名空间
- FormatKey/ParseKey 工具函数
- 命名空间级别 Clear/Keys 操作
```

**版本更新**: MINOR +1

---

### 2. fix - Bug 修复

```bash
git commit -m "fix: 修复缓存并发问题"

- 修复 GetDel 方法中的竞态条件
- 添加写锁保护 LastAccess 字段更新
- 增加并发测试用例覆盖
```

**版本更新**: PATCH +1

---

### 3. perf - 性能优化

```bash
git commit -m "perf: 优化内存分配策略"

- 使用对象池减少 GC 压力
- 批量预分配减少内存碎片
- 基准测试提升 30% 性能
```

**版本更新**: PATCH +1

---

### 4. refactor - 代码重构

```bash
git commit -m "refactor: 重构缓存淘汰策略"

- 提取 EvictionPolicy 接口
- 分离 LRU/LFU 实现到独立文件
- 统一淘汰回调机制
```

**版本更新**: PATCH +1

---

### 5. docs - 文档更新

```bash
git commit -m "docs: 添加技能文档，包括版本管理、结构化日志和TDD"

- 新增 git-version-manager/SKILL.md 文档，描述自动版本管理功能
- 新增 structured-logging/SKILL.md 文档，描述结构化日志系统
- 新增 tdd/SKILL.md 文档，描述测试驱动开发流程
- 所有文档包含详细的功能说明、使用方式和示例
```

**版本更新**: 不更新版本号

---

### 6. chore - 构建/维护任务

```bash
git commit -m "chore: 更新 go.mod 依赖版本"

- 升级 golang.org/x/net 到 v0.17.0
- 更新 golang.org/x/text 到 v0.13.0
- 运行 go mod tidy 清理依赖
```

**版本更新**: 不更新版本号

---

### 7. test - 测试相关

```bash
git commit -m "test: 添加缓存压力测试"

- 新增 benchmark_test.go 性能基准测试
- 添加并发读写压力测试用例
- 验证高并发场景下的稳定性
```

**版本更新**: 不更新版本号

---

### 8. style - 代码格式

```bash
git commit -m "style: 统一代码格式化"

- 运行 gofmt -s 格式化所有源文件
- 调整 import 分组顺序
- 修正注释拼写错误
```

**版本更新**: 不更新版本号

---

### 9. breaking - 破坏性变更

```bash
git commit -m "breaking: 重构 API 接口设计"

- 移除已废弃的 GetTTL 方法
- 更改 Set 方法签名，增加 Options 参数
- 更新所有调用方代码
```

**版本更新**: MAJOR +1

---

## 多模块同时更新示例

当一个提交涉及多个模块时：

```bash
git commit -m "feat: 添加 TCP 服务器和 RESP 协议支持"

server/tcp.go:
- NewTCPServer 创建 TCP 服务器
- 支持 RESP 协议解析
- 兼容 redis-cli 连接

resp/reader.go:
- Read 方法实现 RESP 协议读取
- 支持字符串、整数、数组类型
- 完整的错误处理

cmd/gocache-cli/main.go:
- 交互式命令行工具
- 支持单命令和 REPL 模式
- 自动重连机制
```

**版本更新**: MINOR +1（因为是新增功能）

---

## 版本更新对照表

| 前缀 | 类型 | 版本更新 | 示例 |
|------|------|----------|------|
| feat | 新功能 | MINOR +1 | feat: 添加 XX 功能 |
| fix | Bug 修复 | PATCH +1 | fix: 修复 XX 问题 |
| perf | 性能优化 | PATCH +1 | perf: 优化 XX 性能 |
| refactor | 重构 | PATCH +1 | refactor: 重构 XX |
| docs | 文档 | 不更新 | docs: 更新 XX 文档 |
| chore | 维护 | 不更新 | chore: 更新依赖 |
| test | 测试 | 不更新 | test: 添加 XX 测试 |
| style | 格式 | 不更新 | style: 格式化代码 |
| breaking | 破坏性变更 | MAJOR +1 | breaking: 更改 XX API |

---

## 提交信息模板

### feat 模板
```
feat: 新增[功能名称]

- [模块1]: [具体描述]
- [模块2]: [具体描述]
- [模块3]: [具体描述]
```

### fix 模板
```
fix: 修复[问题描述]

- [问题原因]: [解决方案]
- [影响范围]: [涉及的模块/功能]
- [测试验证]: [添加的测试用例]
```

### perf 模板
```
perf: 优化[优化项]

- [优化前]: [原始实现/性能数据]
- [优化后]: [优化后实现/性能数据]
- [优化方法]: [具体技术手段]
```
