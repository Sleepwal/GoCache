# GoCache

**当前版本: v0.6.0**

一个简单的 Go 语言内存数据库(内存缓存)实现。

## 功能特性

- ✅ 基本的 CRUD 操作 (SET/GET/DELETE)
- ✅ 线程安全(基于 sync.RWMutex)
- ✅ 支持 TTL 过期时间
- ✅ 惰性删除 + 定期清理机制
- ✅ LRU 淘汰策略(最近最少使用)
- ✅ LFU 淘汰策略(最不经常使用，支持时间衰减)
- ✅ 缓存统计指标(命中率、操作计数等)
- ✅ OnEviction 回调函数(缓存项移除时通知)
- ✅ 泛型支持(类型安全的缓存操作)
- ✅ String 操作(Append, Incr/Decr, GetRange 等)
- ✅ List 数据结构(LPUSH/RPUSH, LPOP/RPOP, LRANGE 等)
- ✅ Hash/Map 数据结构(HSET, HGET, HGETALL 等)
- ✅ Set 数据结构(SADD, SREM, SUNION, SINTER 等)
- ✅ 快照/序列化持久化(JSON/Gob 格式)
- ✅ AOF 持久化(Append-Only File)
- ✅ HTTP REST API 服务器
- ✅ 轻量级,无外部依赖
- ✅ 自动版本管理(根据提交信息自动更新版本号和 tag)

## 项目结构

```
GoCache/
├── cache/
│   ├── cache.go                # 核心缓存实现
│   ├── eviction.go             # 过期清理逻辑
│   ├── stats.go                # 统计指标实现
│   ├── generic_cache.go        # 泛型缓存包装器
│   ├── string.go               # String 操作实现
│   ├── list.go                 # List 数据结构实现
│   ├── hash.go                 # Hash 数据结构实现
│   ├── set.go                  # Set 数据结构实现
│   ├── cache_test.go           # 单元测试
│   ├── callback_test.go        # 回调测试
│   ├── stats_test.go           # 统计测试
│   ├── generic_cache_test.go   # 泛型测试
│   ├── string_test.go          # String 测试
│   ├── list_test.go            # List 测试
│   ├── hash_test.go            # Hash 测试
│   ├── set_test.go             # Set 测试
│   ├── lru.go                  # LRU 缓存实现
│   ├── lru_test.go             # LRU 单元测试
│   ├── lfu.go                  # LFU 缓存实现
│   └── lfu_test.go             # LFU 单元测试
├── main.go                     # 示例代码
└── README.md
```

## 快速开始

### 安装

```bash
git clone <repository-url>
cd GoCache
```

### 使用示例

```go
package main

import (
    "fmt"
    "time"

    "GoCache/cache"
)

func main() {
    // 创建普通缓存(永不过期)
    c := cache.New()

    // 创建 LRU 缓存(容量为 100)
    lru := cache.NewLRU(100)

    // 创建 LFU 缓存(容量为 100)
    lfu := cache.NewLFU(100)

    // 设置缓存(永不过期)
    c.Set("name", "GoCache", 0)

    // 设置缓存(带过期时间)
    c.Set("temp", "value", 5*time.Second)

    // 获取缓存
    if value, found := c.Get("name"); found {
        fmt.Println(value) // 输出: GoCache
    }

    // 删除缓存
    c.Delete("temp")

    // 检查键是否存在
    if c.Exists("name") {
        fmt.Println("name exists")
    }

    // 获取所有键
    keys := c.Keys()

    // 清空缓存
    c.Clear()
    
    // 查看统计信息
    stats := c.Stats.GetSnapshot()
    fmt.Printf("Hit Rate: %.2f%%\n", stats.HitRate)
}
```

### OnEviction 回调函数

```go
// 创建带回调的缓存
c := cache.New(cache.WithEvictionCallback(func(key string, value any, reason cache.EvictionReason) {
    fmt.Printf("Key '%s' evicted, reason: %s\n", key, reason)
}))

c.Set("key1", "value1", 50*time.Millisecond)
time.Sleep(100 * time.Millisecond)
c.DeleteExpired() // 触发回调，输出: Key 'key1' evicted, reason: ttl_expired
```

### 泛型缓存(类型安全)

```go
// 创建泛型缓存
c := cache.NewTypedCache[string](cache.New())

c.Set("name", "GoCache", 0)

// Get 返回 string 类型，无需类型断言
if name, found := c.Get("name"); found {
    fmt.Printf("Hello, %s!\n", name)
}

// 也支持自定义结构体
type User struct {
    Name string
    Age  int
}

userCache := cache.NewTypedCache[User](cache.New())
userCache.Set("user1", User{Name: "Alice", Age: 30}, 0)
user, _ := userCache.Get("user1")
fmt.Println(user.Name) // 输出: Alice
```

### LFU 时间衰减

```go
lfu := cache.NewLFU(100)

// 启动定期频率衰减(每 10 秒衰减一次)
stop := lfu.StartDecay(10 * time.Second)
defer stop()

// 设置衰减系数(0.0-1.0，默认 0.5)
lfu.SetDecayFactor(0.8)
```

### String 操作

```go
sc := cache.NewStringCache(cache.New())

// 设置字符串
sc.Set("key", "hello", 0)

// 追加字符串
length := sc.Append("key", " world") // 返回 11

// 整数自增
sc.Set("counter", "10", 0)
val, _ := sc.Incr("counter") // 返回 11

// 获取子字符串
val, _ := sc.GetRange("key", 0, 4) // 返回 "hello"

// 获取字符串长度
length, _ := sc.StrLen("key") // 返回 11
```

### List 数据结构

```go
lc := cache.NewListCache()

// 左侧/右侧推入
lc.LPush("mylist", 0, "a", "b", "c") // [c, b, a]
lc.RPush("mylist", 0, "d")           // [c, b, a, d]

// 左侧/右侧弹出
val, _ := lc.LPop("mylist") // 返回 "c"
val, _ := lc.RPop("mylist") // 返回 "d"

// 范围查询
vals, _ := lc.LRange("mylist", 0, -1) // 返回 [b, a]

// 按索引获取
val, _ := lc.LIndex("mylist", 0) // 返回 "b"

// 修剪列表
lc.LTrim("mylist", 0, 0) // 只保留第一个元素
```

### Hash 数据结构

```go
hc := cache.NewHashCache()

// 设置字段
hc.HSetSingle("user:1", "name", 0, "Alice")
hc.HSet("user:1", 0, map[string]any{
    "age":  30,
    "city": "Beijing",
})

// 获取字段
name, _ := hc.HGet("user:1", "name") // 返回 "Alice"

// 获取所有字段
fields, _ := hc.HGetAll("user:1") // 返回 map[name:Alice age:30 city:Beijing]

// 删除字段
hc.HDel("user:1", "city")

// 字段自增
hc.HSetSingle("user:1", "score", 0, 100)
hc.HIncrBy("user:1", "score", 0, 50) // 返回 150
```

### Set 数据结构

```go
sc := cache.NewSetCache()

// 添加成员
sc.SAdd("myset", 0, "a", "b", "c")

// 检查成员
sc.SIsMember("myset", "a") // 返回 true

// 获取所有成员
members, _ := sc.SMembers("myset")

// 并集/交集/差集
sc.SAdd("set1", 0, "a", "b", "c")
sc.SAdd("set2", 0, "c", "d", "e")

union := sc.SUnion("set1", "set2")   // 返回 [a, b, c, d, e]
inter := sc.SInter("set1", "set2")   // 返回 [c]
diff := sc.SDiff("set1", "set2")     // 返回 [a, b]
```

### HTTP REST API 服务器

```go
package main

import (
    "GoCache/server"
)

func main() {
    // 创建 HTTP 服务器 (默认端口 8080)
    hs := server.NewHTTPServer(server.HTTPServerConfig{
        Port: 8080,
    })

    // 启动服务器
    hs.Start()
}
```

**API 端点:**

```bash
# 设置缓存
curl -X POST http://localhost:8080/cache/mykey \
  -H "Content-Type: application/json" \
  -d '{"value": "myvalue", "ttl": "1h"}'

# 获取缓存
curl http://localhost:8080/cache/mykey

# 删除缓存
curl -X DELETE http://localhost:8080/cache/mykey

# 获取所有键
curl http://localhost:8080/cache/keys

# 获取统计信息
curl http://localhost:8080/cache/stats

# 清空缓存
curl -X POST http://localhost:8080/cache/clear
```

### 定期清理

```go
// 启动定期清理(每 10 秒清理一次过期键)
stop := c.StartEviction(10 * time.Second)

// 停止清理
stop()
```

## API 文档

### 基础缓存 API

#### `New(opts ...Option) *MemoryCache`
创建一个新的内存缓存实例。
- `opts`: 可选配置项(如 `WithEvictionCallback`)

#### `Set(key string, value interface{}, ttl time.Duration)`
添加或更新缓存项。
- `key`: 缓存键
- `value`: 缓存值(任意类型)
- `ttl`: 过期时间,0 表示永不过期

#### `Get(key string) (interface{}, bool)`
获取缓存项。返回值和是否找到的布尔值。

#### `Delete(key string) bool`
删除缓存项。返回是否成功删除。

#### `Exists(key string) bool`
检查键是否存在(包括是否过期)。

#### `Keys() []string`
返回所有未过期的键。

#### `Clear()`
清空所有缓存。

#### `Count() int`
返回缓存项数量(包括已过期的)。

#### `DeleteExpired()`
手动触发清理过期键。

#### `StartEviction(interval time.Duration) func()`
启动定期清理协程,返回停止函数。

#### `Stats *Stats`
统计指标对象。包含:
- `Hits`: 命中次数
- `Misses`: 未命中次数
- `Sets`: 设置次数
- `Deletes`: 删除次数
- `ExpiredCount`: 过期删除次数
- `TTLHits`: TTL 有效期内命中次数
- `TTLMisses`: TTL 过期后未命中次数

### LRU 缓存 API

#### `NewLRU(capacity int, opts ...LRUCacheOption) *LRUCache`
创建一个新的 LRU 缓存。
- `capacity`: 缓存容量，0 表示无限制
- `opts`: 可选配置项(如 `WithLRUEvictionCallback`)

### LFU 缓存 API

#### `NewLFU(capacity int, opts ...LFUCacheOption) *LFUCache`
创建一个新的 LFU 缓存。
- `capacity`: 缓存容量，0 表示无限制
- `opts`: 可选配置项(如 `WithLFUEvictionCallback`)

#### `StartDecay(interval time.Duration) func()`
启动定期频率衰减，返回停止函数。

#### `SetDecayFactor(factor float64)`
设置衰减系数 (0.0 - 1.0)，默认 0.5。

#### `GetFrequencies() map[string]float64`
获取所有键的频率信息(用于调试和监控)。

### 泛型缓存 API

#### `NewTypedCache[T any](cache *MemoryCache) *TypedCache[T]`
创建泛型缓存包装器。

#### `NewTypedLRUCache[T any](capacity int, opts ...LRUCacheOption) *TypedLRUCache[T]`
创建泛型 LRU 缓存。

#### `NewTypedLFUCache[T any](capacity int, opts ...LFUCacheOption) *TypedLFUCache[T]`
创建泛型 LFU 缓存。

### 回调函数 API

#### `EvictionReason` 枚举
- `Manual`: 手动删除
- `TTLExpired`: TTL 过期
- `CapacityEvicted`: 容量淘汰

#### `EvictionCallback func(key string, value any, reason EvictionReason)`
缓存项被移除时的回调函数。

#### `WithEvictionCallback(callback EvictionCallback) Option`
为 MemoryCache 设置回调。

#### `WithLRUEvictionCallback(callback EvictionCallback) LRUCacheOption`
为 LRUCache 设置回调。

#### `WithLFUEvictionCallback(callback EvictionCallback) LFUCacheOption`
为 LFUCache 设置回调。

### 持久化 API

#### `SaveToFile(path string) error`
将缓存状态保存到 JSON 文件。

#### `LoadFromFile(path string) error`
从 JSON 文件恢复缓存状态。

#### `SaveToFileGob(path string) error`
使用 gob 格式保存快照（更高效，仅限 Go 使用）。

#### `LoadFromFileGob(path string) error`
使用 gob 格式加载快照。

#### `NewAOFLogger(path string) (*AOFLogger, error)`
创建 AOF 日志器。

#### `Log(command string, args ...string) error`
记录操作到 AOF 文件。

#### `Replay(cache *MemoryCache) error`
重放 AOF 文件到缓存。

#### `Rewrite(cache *MemoryCache) error`
重写 AOF 文件压缩大小。

#### `Close() error`
关闭 AOF 文件。

### HTTP REST API

#### `NewHTTPServer(cfg HTTPServerConfig) *HTTPServer`
创建 HTTP 缓存服务器。

#### `Start() error`
启动 HTTP 服务器（阻塞）。

#### `StartAsync() <-chan error`
异步启动 HTTP 服务器。

#### `Stop() error`
停止 HTTP 服务器。

**API 端点:**

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/cache/{key}` | 获取缓存 |
| POST | `/cache/{key}` | 设置缓存 |
| DELETE | `/cache/{key}` | 删除缓存 |
| GET | `/cache/keys` | 获取所有键 |
| GET | `/cache/stats` | 获取统计信息 |
| POST | `/cache/clear` | 清空缓存 |

**请求格式:**

```json
{
  "value": "any",
  "ttl": "1h"  // 可选，支持 time.Duration 格式
}
```

**响应格式:**

成功:
```json
{
  "key": "mykey",
  "value": "myvalue"
}
```

错误:
```json
{
  "error": "Not Found",
  "message": "key not found",
  "status": 404
}
```

### String 操作 API

#### `NewStringCache(cache *MemoryCache) *StringCache`
创建 String 类型缓存。

#### `Set(key, value string, ttl time.Duration)`
设置字符串值。

#### `Get(key string) (string, bool)`
获取字符串值。

#### `Append(key, value string) int`
追加字符串到值末尾，返回新长度。

#### `Incr(key string) (int64, error)`
将键的值增加 1。

#### `IncrBy(key string, n int64) (int64, error)`
将键的值增加指定整数。

#### `Decr(key string) (int64, error)`
将键的值减少 1。

#### `DecrBy(key string, n int64) (int64, error)`
将键的值减少指定整数。

#### `GetRange(key string, start, end int) (string, bool)`
获取子字符串（支持负数索引）。

#### `SetRange(key string, offset int, value string) int`
覆盖字符串的指定位置。

#### `StrLen(key string) (int, bool)`
获取字符串长度。

#### `GetSet(key, value string) (string, bool)`
设置新值并返回旧值。

### List 数据结构 API

#### `NewListCache() *ListCache`
创建 List 类型缓存。

#### `LPush(key string, ttl time.Duration, values ...any) int`
从左侧推入一个或多个值，返回新长度。

#### `RPush(key string, ttl time.Duration, values ...any) int`
从右侧推入一个或多个值，返回新长度。

#### `LPop(key string) (any, bool)`
从左侧弹出一个值。

#### `RPop(key string) (any, bool)`
从右侧弹出一个值。

#### `LRange(key string, start, stop int) ([]any, bool)`
获取指定范围的元素（支持负数索引）。

#### `LIndex(key string, index int) (any, bool)`
获取指定索引的元素（支持负数索引）。

#### `LLen(key string) (int, bool)`
获取列表长度。

#### `LTrim(key string, start, stop int) bool`
修剪列表到指定范围。

#### `LRem(key string, count int, value any) int`
删除指定值的元素。

### Hash 数据结构 API

#### `NewHashCache() *HashCache`
创建 Hash 类型缓存。

#### `HSet(key string, ttl time.Duration, fields map[string]any) int`
设置一个或多个字段值，返回新增字段数。

#### `HSetSingle(key, field string, ttl time.Duration, value any) bool`
设置单个字段值，返回是否是新字段。

#### `HGet(key, field string) (any, bool)`
获取字段值。

#### `HGetAll(key string) (map[string]any, bool)`
获取所有字段和值。

#### `HDel(key string, fields ...string) int`
删除一个或多个字段，返回删除数量。

#### `HExists(key, field string) bool`
检查字段是否存在。

#### `HLen(key string) (int, bool)`
获取字段数量。

#### `HKeys(key string) ([]string, bool)`
获取所有字段名。

#### `HVals(key string) ([]any, bool)`
获取所有字段值。

#### `HSetNX(key, field string, ttl time.Duration, value any) bool`
字段不存在时设置值。

#### `HIncrBy(key, field string, ttl time.Duration, n int64) (int64, error)`
将字段的值增加指定整数。

### Set 数据结构 API

#### `NewSetCache() *SetCache`
创建 Set 类型缓存。

#### `SAdd(key string, ttl time.Duration, members ...any) int`
添加一个或多个成员，返回新增数量。

#### `SRem(key string, members ...any) int`
移除一个或多个成员，返回移除数量。

#### `SIsMember(key string, member any) bool`
检查成员是否存在。

#### `SCard(key string) (int, bool)`
获取集合基数（大小）。

#### `SMembers(key string) ([]any, bool)`
获取所有成员。

#### `SPop(key string) (any, bool)`
随机弹出一个成员。

#### `SUnion(keys ...string) []any`
获取多个集合的并集。

#### `SInter(keys ...string) []any`
获取多个集合的交集。

#### `SDiff(key1, key2 string) []any`
获取两个集合的差集（key1 - key2）。

## 运行测试

```bash
go test ./cache -v
```

## 运行示例

```bash
go run main.go
```

## 版本管理

### 自动版本更新

项目提供了自动版本管理脚本,会根据 git 提交信息自动:
1. 更新版本号(语义化版本: 主版本.次版本.补丁)
2. 更新 README 中的版本标记
3. 创建 git tag

### 版本号规则

- **主版本(Major)**: 破坏性更新/BREAKING CHANGE
- **次版本(Minor)**: 新功能/feat
- **补丁版本(Patch)**: Bug 修复、性能优化、重构

### 使用方法

**Windows:**
```bash
# 方式 1: 使用批处理文件
version.bat

# 方式 2: 直接运行 PowerShell 脚本
powershell -ExecutionPolicy Bypass -File scripts/version.ps1
```

**Linux/Mac:**
```bash
chmod +x scripts/version.sh
./scripts/version.sh
```

### 版本提交流程

```bash
# 1. 提交代码变更
git add .
git commit -m "feat: 添加 LRU 淘汰策略"

# 2. 运行版本更新脚本
version.bat  # 或 ./scripts/version.sh

# 3. 推送代码和 tag
git push
git push origin v0.2.0
```

## 技术实现

- **存储结构**: `map[string]*Item`
- **并发控制**: `sync.RWMutex`(读写锁)
- **过期策略**: 惰性删除 + 定期全量清理
- **淘汰策略**: LRU(最近最少使用)
- **数据类型**: 支持任意 `any` 类型

## 后续计划

- [ ] 支持 LFU 淘汰策略
- [ ] 增加持久化功能
- [ ] 提供 HTTP/gRPC 接口
- [ ] 支持数据结构(String, List, Hash, Set)

## License

MIT
