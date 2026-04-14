# GoCache 项目配置

## 可用技能

### 版本管理 (Version Manager)

**触发条件**: 在执行 `git commit` 后

**功能**: 自动根据提交信息更新版本号和创建 git tag

**版本规则**:
- `feat` / 新功能 → 次版本+1 (v0.1.0 → v0.2.0)
- `fix` / `bug` / 修复 → 补丁+1 (v0.1.0 → v0.1.1)
- `perf` / 优化 / 性能 → 补丁+1
- `refactor` / 重构 → 补丁+1
- `BREAKING CHANGE` → 主版本+1 (v0.1.0 → v1.0.0)
- `docs` / `style` / `chore` / `test` → 不更新版本

**工作流程**:
1. 获取当前最新的 git tag 版本
2. 分析提交信息,判断版本升级类型
3. 计算新版本号
4. 更新 README.md 中的版本标记
5. 提交 README 更新
6. 创建 git tag
7. 显示版本更新日志

**详细文档**: 查看 `.qwen/skills/version-manager.md`

## 项目信息

- **项目名称**: GoCache
- **项目类型**: Go 内存数据库
- **当前版本**: v0.3.0
- **Go 版本**: 1.25.4

## 项目结构

```
GoCache/
├── .qwen/
│   └── skills/
│       └── version-manager.md  # 版本管理技能
├── cache/
│   ├── cache.go                # 核心缓存实现
│   ├── eviction.go             # 过期清理逻辑
│   ├── stats.go                # 统计指标实现
│   ├── generic_cache.go        # 泛型缓存包装器
│   ├── cache_test.go           # 单元测试
│   ├── callback_test.go        # 回调测试
│   ├── stats_test.go           # 统计测试
│   ├── generic_cache_test.go   # 泛型测试
│   ├── lru.go                  # LRU 缓存实现
│   ├── lru_test.go             # LRU 单元测试
│   ├── lfu.go                  # LFU 缓存实现
│   └── lfu_test.go             # LFU 单元测试
├── main.go                     # 示例程序
├── README.md                   # 项目文档
└── go.mod
```

## 功能特性

- ✅ 基本的 CRUD 操作 (SET/GET/DELETE)
- ✅ 线程安全 (基于 sync.RWMutex)
- ✅ 支持 TTL 过期时间
- ✅ 惰性删除 + 定期清理机制
- ✅ LRU 淘汰策略 (最近最少使用)
- ✅ LFU 淘汰策略 (最不经常使用，支持时间衰减)
- ✅ 缓存统计指标 (命中率、操作计数等)
- ✅ OnEviction 回调函数 (缓存项移除时通知)
- ✅ 泛型支持 (类型安全的缓存操作)
