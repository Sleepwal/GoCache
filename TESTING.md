# GoCache 测试文档

本文档提供完整的测试示例，帮助你快速测试 GoCache 的所有功能。

## 目录

1. [环境准备](#环境准备)
2. [HTTP REST API 测试](#http-rest-api-测试)
   - [基本缓存操作](#1-基本缓存操作)
   - [统计与管理](#2-统计与管理)
3. [Go 代码测试示例](#go-代码测试示例)
   - [String 操作](#3-string-操作)
   - [List 数据结构](#4-list-数据结构)
   - [Hash 数据结构](#5-hash-数据结构)
   - [Set 数据结构](#6-set-数据结构)
   - [泛型缓存](#7-泛型缓存)
   - [持久化](#8-持久化)
   - [命名空间](#9-命名空间)
   - [发布/订阅](#10-发布订阅)

---

## 环境准备

### 1. 启动 HTTP 服务器

```bash
# 克隆并进入项目目录
git clone <repository-url>
cd GoCache

# 启动服务器（默认端口 8080）
go run main.go
```

启动后你会看到类似以下输出：
```
Starting HTTP server on :8080...
API Endpoints:
  GET    /cache/{key}    - Get cache value
  POST   /cache/{key}    - Set cache value
  DELETE /cache/{key}    - Delete cache
  GET    /cache/keys     - Get all keys
  GET    /cache/stats    - Get cache statistics
  POST   /cache/clear    - Clear all cache

Press Ctrl+C to stop the server.
```

---

## HTTP REST API 测试

### 1. 基本缓存操作

#### 设置缓存

```bash
# 设置字符串缓存
curl -X POST http://localhost:8080/cache/name \
  -H "Content-Type: application/json" \
  -d '{"value": "GoCache", "ttl": "1h"}'
```

**响应:**
```json
{
  "key": "name",
  "message": "cache set successfully",
  "ttl": "1h0m0s"
}
```

#### 获取缓存

```bash
curl http://localhost:8080/cache/name
```

**响应:**
```json
{
  "key": "name",
  "value": "GoCache"
}
```

#### 获取不存在的键

```bash
curl http://localhost:8080/cache/nonexistent
```

**响应:**
```json
{
  "error": "Not Found",
  "message": "key not found",
  "status": 404
}
```

#### 删除缓存

```bash
curl -X DELETE http://localhost:8080/cache/name
```

**响应:**
```json
{
  "key": "name",
  "message": "cache deleted successfully"
}
```

#### 删除不存在的键

```bash
curl -X DELETE http://localhost:8080/cache/nonexistent
```

**响应:**
```json
{
  "error": "Not Found",
  "message": "key not found",
  "status": 404
}
```

### 2. 统计与管理

#### 获取所有键

```bash
# 先设置几个键
curl -X POST http://localhost:8080/cache/key1 \
  -H "Content-Type: application/json" \
  -d '{"value": "value1"}'

curl -X POST http://localhost:8080/cache/key2 \
  -H "Content-Type: application/json" \
  -d '{"value": "value2"}'

# 获取所有键
curl http://localhost:8080/cache/keys
```

**响应:**
```json
{
  "keys": ["name", "key1", "key2"],
  "count": 3
}
```

#### 获取统计信息

```bash
curl http://localhost:8080/cache/stats
```

**响应:**
```json
{
  "deletes": 2,
  "expired": 0,
  "hit_rate": "75.00%",
  "hits": 3,
  "misses": 2,
  "sets": 5,
  "total_ops": 12,
  "ttl_hits": 2,
  "ttl_misses": 1
}
```

#### 清空缓存

```bash
curl -X POST http://localhost:8080/cache/clear
```

**响应:**
```json
{
  "message": "cache cleared successfully"
}
```

#### 验证清空结果

```bash
curl http://localhost:8080/cache/keys
```

**响应:**
```json
{
  "count": 0,
  "keys": []
}
```

---

## Go 代码测试示例

对于高级功能（List/Hash/Set/PubSub等），需要使用 Go 代码测试。

### 3. String 操作

创建文件 `test/string_test.go` 并运行：

```go
package main

import (
    "fmt"
    "GoCache/cache"
)

func main() {
    sc := cache.NewStringCache(cache.New())

    // 设置字符串
    sc.Set("key", "hello", 0)
    fmt.Println("Set key:", "hello")

    // 追加字符串
    length := sc.Append("key", " world")
    fmt.Printf("Append ' world', new length: %d\n", length)

    // 获取结果
    val, _ := sc.Get("key")
    fmt.Printf("Get key: %v\n", val)

    // 获取子字符串
    sub, _ := sc.GetRange("key", 0, 4)
    fmt.Printf("GetRange(0, 4): %v\n", sub)

    // 整数自增
    sc.Set("counter", "10", 0)
    val, _ = sc.Incr("counter")
    fmt.Printf("Incr counter: %v\n", val)

    // 字符串长度
    length, _ = sc.StrLen("key")
    fmt.Printf("StrLen: %d\n", length)
}
```

### 4. List 数据结构

```go
package main

import (
    "fmt"
    "GoCache/cache"
)

func main() {
    lc := cache.NewListCache()

    // 左侧推入
    lc.LPush("mylist", 0, "a", "b", "c")
    fmt.Println("LPush 'a', 'b', 'c'")

    // 获取所有元素
    vals, _ := lc.LRange("mylist", 0, -1)
    fmt.Printf("LRange: %v\n", vals)

    // 左侧弹出
    val, _ := lc.LPop("mylist")
    fmt.Printf("LPop: %v\n", val)

    // 获取剩余元素
    vals, _ = lc.LRange("mylist", 0, -1)
    fmt.Printf("After LPop: %v\n", vals)

    // 列表长度
    length, _ := lc.LLen("mylist")
    fmt.Printf("LLen: %d\n", length)
}
```

### 5. Hash 数据结构

```go
package main

import (
    "fmt"
    "GoCache/cache"
)

func main() {
    hc := cache.NewHashCache()

    // 设置字段
    hc.HSetSingle("user:1", "name", 0, "Alice")
    hc.HSetSingle("user:1", "age", 0, 30)
    hc.HSetSingle("user:1", "city", 0, "Beijing")
    fmt.Println("HSet name=Alice, age=30, city=Beijing")

    // 获取单个字段
    name, _ := hc.HGet("user:1", "name")
    fmt.Printf("HGet name: %v\n", name)

    // 获取所有字段
    fields, _ := hc.HGetAll("user:1")
    fmt.Printf("HGetAll: %v\n", fields)

    // 字段数量
    count, _ := hc.HLen("user:1")
    fmt.Printf("HLen: %d\n", count)

    // 删除字段
    hc.HDel("user:1", "city")
    fmt.Println("HDel city")

    fields, _ = hc.HGetAll("user:1")
    fmt.Printf("After HDel: %v\n", fields)
}
```

### 6. Set 数据结构

```go
package main

import (
    "fmt"
    "GoCache/cache"
)

func main() {
    sc := cache.NewSetCache()

    // 添加成员
    sc.SAdd("myset", 0, "a", "b", "c")
    fmt.Println("SAdd 'a', 'b', 'c'")

    // 检查成员
    isMember := sc.SIsMember("myset", "a")
    fmt.Printf("SIsMember 'a': %v\n", isMember)

    // 获取所有成员
    members, _ := sc.SMembers("myset")
    fmt.Printf("SMembers: %v\n", members)

    // 集合基数（大小）
    card, _ := sc.SCard("myset")
    fmt.Printf("SCard: %d\n", card)

    // 并集/交集/差集
    sc.SAdd("set1", 0, "a", "b", "c")
    sc.SAdd("set2", 0, "c", "d", "e")

    union := sc.SUnion("set1", "set2")
    fmt.Printf("SUnion: %v\n", union)

    inter := sc.SInter("set1", "set2")
    fmt.Printf("SInter: %v\n", inter)

    diff := sc.SDiff("set1", "set2")
    fmt.Printf("SDiff: %v\n", diff)
}
```

### 7. 泛型缓存

```go
package main

import (
    "fmt"
    "GoCache/cache"
)

func main() {
    // 泛型字符串缓存
    strCache := cache.NewTypedCache[string](cache.New())
    strCache.Set("name", "GoCache", 0)
    name, found := strCache.Get("name")
    fmt.Printf("TypedCache[string]: %v, found: %v\n", name, found)

    // 泛型整数缓存
    intCache := cache.NewTypedCache[int](cache.New())
    intCache.Set("counter", 100, 0)
    counter, found := intCache.Get("counter")
    fmt.Printf("TypedCache[int]: %v, found: %v\n", counter, found)

    // 自定义结构体
    type User struct {
        Name string
        Age  int
    }
    userCache := cache.NewTypedCache[User](cache.New())
    userCache.Set("user1", User{Name: "Alice", Age: 30}, 0)
    user, found := userCache.Get("user1")
    fmt.Printf("TypedCache[User]: %+v, found: %v\n", user, found)
}
```

### 8. 持久化

```go
package main

import (
    "fmt"
    "GoCache/cache"
)

func main() {
    c := cache.New()
    
    // 设置一些缓存
    c.Set("name", "GoCache", 0)
    c.Set("version", "0.7.0", 0)
    
    // 保存到文件
    err := c.SaveToFile("/tmp/cache_backup.json")
    if err != nil {
        fmt.Printf("SaveToFile error: %v\n", err)
        return
    }
    fmt.Println("Saved cache to /tmp/cache_backup.json")
    
    // 创建新缓存并从文件加载
    c2 := cache.New()
    err = c2.LoadFromFile("/tmp/cache_backup.json")
    if err != nil {
        fmt.Printf("LoadFromFile error: %v\n", err)
        return
    }
    
    // 验证数据
    name, _ := c2.Get("name")
    version, _ := c2.Get("version")
    fmt.Printf("Loaded: name=%v, version=%v\n", name, version)
}
```

### 9. 命名空间

```go
package main

import (
    "fmt"
    "GoCache/cache"
)

func main() {
    c := cache.New()
    
    // 创建命名空间
    userNS := cache.NewNamespaceCache(c, "user")
    orderNS := cache.NewNamespaceCache(c, "order")
    
    // 设置数据（不同命名空间的同名键互不影响）
    userNS.Set("key1", "user_value", 0)
    orderNS.Set("key1", "order_value", 0)
    
    // 获取数据
    userVal, _ := userNS.Get("key1")
    orderVal, _ := orderNS.Get("key1")
    fmt.Printf("user:key1 = %v\n", userVal)
    fmt.Printf("order:key1 = %v\n", orderVal)
    
    // 获取命名空间下的键
    userKeys := userNS.Keys()
    orderKeys := orderNS.Keys()
    fmt.Printf("userNS.Keys(): %v\n", userKeys)
    fmt.Printf("orderNS.Keys(): %v\n", orderKeys)
    
    // 清空单个命名空间
    deleted := userNS.Clear()
    fmt.Printf("Cleared %d keys from user namespace\n", deleted)
    
    // 验证 order 未受影响
    orderKeys = orderNS.Keys()
    fmt.Printf("After clear, orderNS.Keys(): %v\n", orderKeys)
}
```

### 10. 发布/订阅

```go
package main

import (
    "fmt"
    "time"
    "GoCache/cache"
)

func main() {
    pc := cache.NewPubSubCache(cache.New())
    
    // 订阅特定键
    sub := pc.Subscribe("key1")
    
    // 启动接收器
    done := make(chan bool)
    go func() {
        for {
            select {
            case event := <-sub.Channel():
                fmt.Printf("Received event: Type=%s, Key=%s\n", 
                    event.Type, event.Key)
            case <-time.After(2 * time.Second):
                done <- true
                return
            }
        }
    }()
    
    // 发布事件（通过 Set/Delete 操作）
    pc.Set("key1", "value1", 0)
    time.Sleep(100 * time.Millisecond)
    
    pc.Delete("key1")
    time.Sleep(100 * time.Millisecond)
    
    // 等待接收完成
    <-done
    
    // 取消订阅
    pc.Unsubscribe(sub)
    fmt.Println("Unsubscribed")
}
```

---

## 完整测试脚本

以下是一个完整的 shell 脚本，可以自动化测试所有 HTTP API：

```bash
#!/bin/bash

# GoCache HTTP API 测试脚本
# 使用方法: chmod +x test_api.sh && ./test_api.sh

BASE_URL="http://localhost:8080/cache"
PASS=0
FAIL=0

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo "=== GoCache HTTP API 测试 ==="
echo ""

# 测试函数
test_api() {
    local test_name=$1
    local expected_status=$2
    local actual_status=$3
    
    if [ "$actual_status" == "$expected_status" ]; then
        echo -e "${GREEN}✓ PASS${NC}: $test_name"
        PASS=$((PASS+1))
    else
        echo -e "${RED}✗ FAIL${NC}: $test_name (expected $expected_status, got $actual_status)"
        FAIL=$((FAIL+1))
    fi
}

# 1. 设置缓存
echo "1. 设置缓存"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/test1" \
  -H "Content-Type: application/json" \
  -d '{"value": "hello", "ttl": "1h"}')
test_api "Set cache" "201" "$STATUS"

# 2. 获取缓存
echo "2. 获取缓存"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/test1")
test_api "Get cache" "200" "$STATUS"

# 3. 获取不存在的键
echo "3. 获取不存在的键"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/nonexistent")
test_api "Get non-existent" "404" "$STATUS"

# 4. 删除缓存
echo "4. 删除缓存"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/test1")
test_api "Delete cache" "200" "$STATUS"

# 5. 删除不存在的键
echo "5. 删除不存在的键"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/test1")
test_api "Delete non-existent" "404" "$STATUS"

# 6. 获取所有键
echo "6. 获取所有键"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/keys")
test_api "Get keys" "200" "$STATUS"

# 7. 获取统计信息
echo "7. 获取统计信息"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/stats")
test_api "Get stats" "200" "$STATUS"

# 8. 清空缓存
echo "8. 清空缓存"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/clear")
test_api "Clear cache" "200" "$STATUS"

# 总结
echo ""
echo "=== 测试结果 ==="
echo -e "通过: ${GREEN}$PASS${NC}"
echo -e "失败: ${RED}$FAIL${NC}"
echo "总计: $((PASS+FAIL))"

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}全部测试通过！${NC}"
else
    echo -e "${RED}部分测试失败${NC}"
fi
```

运行脚本：

```bash
chmod +x test_api.sh
./test_api.sh
```

---

## 性能基准测试

GoCache 内置了性能基准测试：

```bash
# 运行基准测试
go test ./cache -bench=. -benchtime=1s

# 示例输出:
# BenchmarkMemoryCache_Set-4       1568960    639.3 ns/op
# BenchmarkMemoryCache_Get-4      15615259     75.51 ns/op
# BenchmarkLRUCache_Set-4          3735482    314.3 ns/op
# BenchmarkLFUCache_Set-4          5093146    241.7 ns/op
```

---

## 常见问题

### Q: 服务器启动失败？
确保 8080 端口未被占用，或修改 `main.go` 中的端口号。

### Q: curl 请求返回 405 Method Not Allowed？
检查请求方法是否正确，例如：
- GET 用于获取
- POST 用于设置
- DELETE 用于删除

### Q: 如何查看更详细的调试信息？
运行测试时添加 `-v` 参数：
```bash
go test ./cache -v
go test ./server -v
```
