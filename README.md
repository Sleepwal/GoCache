# GoCache

一个简单的 Go 语言内存数据库(内存缓存)实现。

## 功能特性

- ✅ 基本的 CRUD 操作 (SET/GET/DELETE)
- ✅ 线程安全(基于 sync.RWMutex)
- ✅ 支持 TTL 过期时间
- ✅ 惰性删除 + 定期清理机制
- ✅ 轻量级,无外部依赖

## 项目结构

```
GoCache/
├── cache/
│   ├── cache.go          # 核心缓存实现
│   ├── eviction.go       # 过期清理逻辑
│   └── cache_test.go     # 单元测试
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
    // 创建缓存
    c := cache.New()
    
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
}
```

### 定期清理

```go
// 启动定期清理(每 10 秒清理一次过期键)
stop := c.StartEviction(10 * time.Second)

// 停止清理
stop()
```

## API 文档

### `New() *MemoryCache`
创建一个新的内存缓存实例。

### `Set(key string, value interface{}, ttl time.Duration)`
添加或更新缓存项。
- `key`: 缓存键
- `value`: 缓存值(任意类型)
- `ttl`: 过期时间,0 表示永不过期

### `Get(key string) (interface{}, bool)`
获取缓存项。返回值和是否找到的布尔值。

### `Delete(key string) bool`
删除缓存项。返回是否成功删除。

### `Exists(key string) bool`
检查键是否存在(包括是否过期)。

### `Keys() []string`
返回所有未过期的键。

### `Clear()`
清空所有缓存。

### `Count() int`
返回缓存项数量(包括已过期的)。

### `DeleteExpired()`
手动触发清理过期键。

### `StartEviction(interval time.Duration) func()`
启动定期清理协程,返回停止函数。

## 运行测试

```bash
go test ./cache -v
```

## 运行示例

```bash
go run main.go
```

## 技术实现

- **存储结构**: `map[string]*Item`
- **并发控制**: `sync.RWMutex`(读写锁)
- **过期策略**: 惰性删除 + 定期全量清理
- **数据类型**: 支持任意 `interface{}` 类型

## 后续计划

- [ ] 支持 LRU/LFU 淘汰策略
- [ ] 增加持久化功能
- [ ] 提供 HTTP/gRPC 接口
- [ ] 支持数据结构(String, List, Hash, Set)

## License

MIT
