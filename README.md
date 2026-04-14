# GoCache

**当前版本: v0.3.0**

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
- ✅ 轻量级,无外部依赖
- ✅ 自动版本管理(根据提交信息自动更新版本号和 tag)

## 项目结构

```
GoCache/
├── cache/
│   ├── cache.go          # 核心缓存实现
│   ├── eviction.go       # 过期清理逻辑
│   ├── stats.go          # 统计指标实现
│   ├── generic_cache.go  # 泛型缓存包装器
│   ├── cache_test.go     # 单元测试
│   ├── callback_test.go  # 回调测试
│   ├── stats_test.go     # 统计测试
│   ├── generic_cache_test.go # 泛型测试
│   ├── lru.go            # LRU 缓存实现
│   ├── lru_test.go       # LRU 单元测试
│   ├── lfu.go            # LFU 缓存实现
│   └── lfu_test.go       # LFU 单元测试
├── main.go               # 示例代码
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
