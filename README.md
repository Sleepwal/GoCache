# GoCache

**当前版本: v0.1.0**

一个简单的 Go 语言内存数据库(内存缓存)实现。

## 功能特性

- ✅ 基本的 CRUD 操作 (SET/GET/DELETE)
- ✅ 线程安全(基于 sync.RWMutex)
- ✅ 支持 TTL 过期时间
- ✅ 惰性删除 + 定期清理机制
- ✅ 轻量级,无外部依赖
- ✅ 自动版本管理(根据提交信息自动更新版本号和 tag)

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
- **数据类型**: 支持任意 `interface{}` 类型

## 后续计划

- [ ] 支持 LRU/LFU 淘汰策略
- [ ] 增加持久化功能
- [ ] 提供 HTTP/gRPC 接口
- [ ] 支持数据结构(String, List, Hash, Set)

## License

MIT
