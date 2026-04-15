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
4. 更新 README.md 和 QWEN.md 中的版本标记
5. 提交 README 和 QWEN.md 更新
6. 创建 git tag
7. 显示版本更新日志

**详细文档**: 查看 `.qwen/skills/git-version-manager.md`

## 项目信息

- **项目名称**: GoCache
- **项目类型**: Go 内存数据库
- **当前版本**: v1.0.0
- **Go 版本**: 1.25.4

## 项目结构

```
GoCache/
├── .qwen/
│   └── skills/
│       └── git-version-manager.md  # 版本管理技能
├── cache/
│   ├── cache.go                # 核心缓存实现
│   ├── eviction.go             # 过期清理逻辑
│   ├── stats.go                # 统计指标实现
│   ├── generic_cache.go        # 泛型缓存包装器
│   ├── string.go               # String 操作实现
│   ├── list.go                 # List 数据结构实现
│   ├── hash.go                 # Hash 数据结构实现
│   ├── set.go                  # Set 数据结构实现
│   ├── snapshot.go             # 快照/序列化实现
│   ├── aof.go                  # AOF 持久化实现
│   ├── namespace.go            # 缓存命名空间实现
│   ├── pubsub.go               # 发布/订阅系统实现
│   ├── benchmark_test.go       # 性能基准测试
│   ├── memory_test.go          # 内存限制测试
│   ├── namespace_test.go       # 命名空间测试
│   ├── pubsub_test.go          # Pub/Sub 测试
│   ├── cache_test.go           # 单元测试
│   ├── callback_test.go        # 回调测试
│   ├── stats_test.go           # 统计测试
│   ├── generic_cache_test.go   # 泛型测试
│   ├── string_test.go          # String 测试
│   ├── list_test.go            # List 测试
│   ├── hash_test.go            # Hash 测试
│   ├── set_test.go             # Set 测试
│   ├── snapshot_test.go        # 快照测试
│   ├── aof_test.go             # AOF 测试
│   ├── lru.go                  # LRU 缓存实现
│   ├── lru_test.go             # LRU 单元测试
│   ├── lfu.go                  # LFU 缓存实现
│   └── lfu_test.go             # LFU 单元测试
├── server/
│   ├── http.go                 # HTTP REST API 服务器
│   └── http_test.go            # HTTP 服务器测试
├── main.go                     # 示例程序
├── README.md                   # 项目文档
├── QWEN.md                     # 项目配置文件
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
- ✅ String 操作 (Append, Incr/Decr, GetRange 等)
- ✅ List 数据结构 (LPUSH/RPUSH, LPOP/RPOP, LRANGE 等)
- ✅ Hash/Map 数据结构 (HSET, HGET, HGETALL 等)
- ✅ Set 数据结构 (SADD, SREM, SUNION, SINTER 等)
- ✅ 快照/序列化持久化 (JSON/Gob 格式)
- ✅ AOF 持久化 (Append-Only File)
- ✅ HTTP REST API 服务器
- ✅ 性能基准测试套件
- ✅ 内存限制
- ✅ 缓存命名空间
- ✅ 发布/订阅系统
