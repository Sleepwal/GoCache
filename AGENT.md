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
4. 更新 README.md 和 AGENT.md 中的版本标记
5. 提交 README 和 AGENT.md 更新
6. 创建 git tag
7. 显示版本更新日志

**详细文档**: 查看 `.trae/skills/git-version-manager/SKILL.md`

## 项目信息

- **项目名称**: GoCache
- **项目类型**: Go 内存数据库
- **当前版本**: v2.0.0
- **Go 版本**: 1.21+

## 项目结构

```
GoCache/
├── .trae/
│   └── skills/
│       ├── git-version-manager/
│       │   └── SKILL.md         # 版本管理技能
│       ├── structured-logging/
│       │   └── SKILL.md         # 结构化日志技能
│       └── tdd/
│           └── SKILL.md         # TDD测试驱动开发技能
├── cache/
│   ├── cache.go                 # 核心缓存实现
│   ├── eviction.go              # 过期清理逻辑
│   ├── stats.go                 # 统计指标实现
│   ├── generic_cache.go         # 泛型缓存包装器
│   ├── string.go                # String 操作实现
│   ├── list.go                  # List 数据结构实现
│   ├── hash.go                  # Hash 数据结构实现
│   ├── set.go                   # Set 数据结构实现
│   ├── sorted_set.go            # Sorted Set 数据结构实现
│   ├── snapshot.go              # 快照/序列化实现
│   ├── aof.go                   # AOF 持久化实现
│   ├── namespace.go             # 缓存命名空间实现
│   ├── pubsub.go                # 发布/订阅系统实现
│   ├── lru.go                   # LRU 缓存实现
│   ├── lfu.go                   # LFU 缓存实现
│   └── *_test.go                # 对应单元测试
├── server/
│   ├── http.go                  # HTTP REST API 服务器
│   ├── http_test.go             # HTTP 服务器测试
│   ├── tcp.go                   # TCP 服务器 (RESP 协议)
│   └── tcp_test.go              # TCP 服务器测试
├── resp/
│   ├── reader.go                # RESP 协议读取器
│   ├── writer.go                # RESP 协议写入器
│   └── reader_test.go           # RESP 读取器测试
├── logger/
│   ├── logger.go                # 日志核心实现
│   ├── config.go                # 日志配置
│   ├── writer.go                # 日志写入器
│   ├── global.go                # 全局日志接口
│   └── *_test.go                # 日志测试
├── cmd/
│   └── gocache-cli/
│       └── main.go              # 命令行工具
├── main.go                      # 主程序入口
├── AGENT.md                     # 项目配置文件
├── README.md                     # 项目文档
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
- ✅ Sorted Set 数据结构 (ZADD, ZRANGE, ZSCORE 等)
- ✅ 快照/序列化持久化 (JSON/Gob 格式)
- ✅ AOF 持久化 (Append-Only File)
- ✅ HTTP REST API 服务器
- ✅ TCP 服务器 (RESP 协议，兼容 Redis)
- ✅ gocache-cli 命令行工具
- ✅ 性能基准测试套件
- ✅ 内存限制
- ✅ 缓存命名空间
- ✅ 发布/订阅系统
- ✅ 结构化日志系统
- ✅ 轻量级，无外部依赖

## API 端点

### HTTP API (端口 8080)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/cache/{key}` | 获取缓存 |
| POST | `/cache/{key}` | 设置缓存 |
| DELETE | `/cache/{key}` | 删除缓存 |
| GET | `/cache/keys` | 获取所有键 |
| GET | `/cache/stats` | 获取统计信息 |
| POST | `/cache/clear` | 清空缓存 |
| GET | `/cache/health` | 健康检查 |
| POST | `/cache/string/{key}` | String 操作 |
| POST | `/cache/list/{key}` | List 操作 |
| POST | `/cache/hash/{key}` | Hash 操作 |
| POST | `/cache/set/{key}` | Set 操作 |
| POST | `/cache/zset/{key}` | Sorted Set 操作 |

### RESP 协议 (端口 6379)

兼容 Redis 协议，支持以下命令：
- 基础: GET, SET, DEL, EXISTS, KEYS, FLUSHDB
- String: APPEND, INCR, DECR, STRLEN
- List: LPUSH, RPUSH, LPOP, RPOP, LRANGE, LEN
- Hash: HSET, HGET, HGETALL, HDEL
- Set: SADD, SREM, SMEMBERS, SISMEMBER
- Sorted Set: ZADD, ZRANGE, ZSCORE, ZRANK

### 命令行工具

```bash
gocache-cli -h 127.0.0.1 -p 6379
```

## 开发规范

### 代码风格
- 使用 `gofmt` 格式化代码
- 使用 `golangci-lint` 进行代码检查
- 遵循 Go 官方代码评审建议

### 测试规范
- 单元测试覆盖率目标 > 80%
- 使用表格驱动测试方法
- 性能基准测试使用 `testing.B`

### 提交规范
- 提交信息使用中文
- 使用语义化提交前缀:
  - `feat:` 新功能
  - `fix:` Bug 修复
  - `perf:` 性能优化
  - `refactor:` 重构
  - `docs:` 文档
  - `chore:` 构建/工具
  - `test:` 测试

## 后续计划

- [ ] gRPC 服务接口
- [ ] 集群/分布式支持
- [ ] Web 管理界面
- [ ] 更多数据结构 (Stream, Geo 等)
- [ ] 集群支持 (Redis Cluster 协议)

## License

本项目采用 [GNU General Public License v3.0](LICENSE) 许可。
