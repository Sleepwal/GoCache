# TDD (测试驱动开发)

## 触发条件
当用户明确要求使用TDD方式开发时触发：
- "使用TDD开发XXX功能"
- "用测试驱动的方式实现XXX"
- "先写测试再实现XXX"
- "TDD: 实现XXX功能"

用户也可以在日常开发中主动要求：
- "为XXX功能编写测试"
- "使用TDD流程重构XXX"

## 功能说明
使用测试驱动开发（TDD）方法论进行软件开发的标准化流程。TDD 的核心理念是**先写测试，再写实现**，确保代码质量和可维护性。

## TDD 核心循环（Red-Green-Refactor）

### 🔴 Red（红灯）- 编写失败的测试
1. **理解需求**：明确要实现的功能和行为
2. **编写测试**：根据需求编写一个测试用例
3. **运行测试**：确保测试失败（这是预期的）
4. **验证测试**：确认测试失败的原因是正确的

### 🟢 Green（绿灯）- 快速实现通过测试的代码
1. **实现功能**：编写最少量的代码让测试通过
2. **不追求完美**：可以先用最简单的实现（甚至硬编码）
3. **运行测试**：确保测试通过
4. **不要重构**：此时不要修改代码结构

### 🔵 Refactor（蓝灯）- 重构代码
1. **优化代码**：在测试保护下安全重构
2. **消除重复**：提取公共逻辑，改善设计
3. **运行测试**：确保重构后测试仍然通过
4. **重复循环**：继续下一个测试用例

## 执行流程

### 阶段1: 需求分析
```
用户: "使用TDD实现一个支持过期时间的缓存功能"

AI助手:
1. 分析需求，拆解为具体的行为：
   - 设置带TTL的键值对
   - 获取未过期的键值对
   - 获取已过期的键值对应返回不存在
   - 自动清理过期键
   
2. 设计测试用例清单
3. 与用户确认测试计划
```

### 阶段2: Red - 编写测试
```go
// cache_test.go
func TestCache_SetWithTTL(t *testing.T) {
    c := New()
    c.Set("key", "value", 1*time.Second)
    
    // 立即获取应该成功
    val, found := c.Get("key")
    if !found || val != "value" {
        t.Errorf("expected 'value', got %v, found=%v", val, found)
    }
    
    // 等待过期
    time.Sleep(2 * time.Second)
    
    // 过期后获取应失败
    _, found = c.Get("key")
    if found {
        t.Error("expected key to be expired")
    }
}
```

### 阶段3: Green - 实现代码
```go
// cache.go
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
    c.items[key] = Item{
        Value:     value,
        ExpiresAt: time.Now().Add(ttl),
    }
}

func (c *Cache) Get(key string) (interface{}, bool) {
    item, exists := c.items[key]
    if !exists {
        return nil, false
    }
    if time.Now().After(item.ExpiresAt) {
        delete(c.items, key)
        return nil, false
    }
    return item.Value, true
}
```

### 阶段4: Refactor - 重构优化
- 提取过期检查逻辑为独立方法
- 改善代码结构和可读性
- 确保所有测试仍然通过

### 阶段5: 重复循环
- 编写下一个测试
- 重复 Red-Green-Refactor 循环

## TDD 工作规范

### 1. 测试先行原则
- **必须先写测试**：在编写任何实现代码之前，先编写测试用例
- **测试要失败**：确保新编写的测试在未实现时确实失败
- **测试要明确**：测试失败信息应该清晰指出问题所在

### 2. 最小实现原则
- **刚好通过测试**：只编写让测试通过的最少代码
- **避免过度设计**：不要提前实现未测试的功能
- **允许硬编码**：初期可以用硬编码快速通过测试

### 3. 重构安全原则
- **测试全覆盖**：重构前确保有足够的测试覆盖
- **小步快跑**：每次只做小的重构改动
- **随时可回退**：每次重构后运行测试确保通过

### 4. 测试质量要求
- **测试独立性**：每个测试应该独立运行，不依赖其他测试
- **测试可重复**：测试结果应该一致，不受外部因素影响
- **测试有意义**：测试名称要清晰表达测试意图
- **测试边界清晰**：每个测试只验证一个行为

## Go 语言 TDD 实践

### 测试文件组织
```
project/
├── cache/
│   ├── cache.go          # 实现代码
│   └── cache_test.go     # 测试代码（与实现同目录）
└── main.go
```

### 测试函数命名规范
```go
// 格式: Test{功能}_{场景}_{预期结果}
func TestCache_SetAndGet_ExistingKey_ReturnsValue(t *testing.T) { }
func TestCache_Get_NonExistentKey_ReturnsFalse(t *testing.T) { }
func TestCache_SetWithTTL_ExpiredKey_ReturnsFalse(t *testing.T) { }
```

### 表格驱动测试（推荐）
```go
func TestCache_SetAndGet(t *testing.T) {
    tests := []struct {
        name      string
        key       string
        value     interface{}
        ttl       time.Duration
        wait      time.Duration
        wantFound bool
    }{
        {"永久键", "key1", "value1", 0, 0, true},
        {"短期TTL键", "key2", "value2", 2*time.Second, 1*time.Second, true},
        {"已过期TTL键", "key3", "value3", 1*time.Second, 2*time.Second, false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            c := New()
            c.Set(tt.key, tt.value, tt.ttl)
            if tt.wait > 0 {
                time.Sleep(tt.wait)
            }
            _, found := c.Get(tt.key)
            if found != tt.wantFound {
                t.Errorf("expected found=%v, got %v", tt.wantFound, found)
            }
        })
    }
}
```

### Mock 和测试替身
```go
// 定义接口
type EvictionPolicy interface {
    ShouldEvict(key string) bool
}

// 使用接口
type Cache struct {
    policy EvictionPolicy
    // ...
}

// 测试时注入Mock
func TestCache_WithMockPolicy(t *testing.T) {
    mockPolicy := &MockEvictionPolicy{}
    c := NewWithPolicy(mockPolicy)
    // ...
}
```

## TDD 开发节奏

### 单个循环建议时间
- **Red (写测试)**: 5-10分钟
- **Green (实现)**: 5-15分钟  
- **Refactor (重构)**: 5-10分钟
- **总时长**: 15-35分钟/循环

### 循环迭代策略
1. **从简单开始**：先测试最基本的使用场景
2. **逐步增加复杂度**：每次添加一个新场景
3. **覆盖边界情况**：正常流程测试完后测试边界
4. **处理异常情况**：最后测试错误处理

### 测试覆盖优先级
1. **核心业务逻辑** - 最高优先级
2. **公共API** - 必须覆盖
3. **边界条件** - 重要
4. **错误处理** - 必要
5. **性能测试** - 视情况而定

## 常见场景处理

### 场景1: 新功能开发
```
用户: "使用TDD实现LRU缓存"

流程:
1. 分析LRU需求，列出行为清单
2. 编写 TestLRUCache_SetAndGet 测试
3. 实现最基础的 Set/Get
4. 运行测试，确保通过
5. 编写 TestLRUCache_Eviction 测试
6. 实现淘汰逻辑
7. 重构优化
8. 继续下一个测试
```

### 场景2: Bug修复
```
用户: "修复缓存并发问题，使用TDD"

流程:
1. 编写复现Bug的测试
2. 确保测试失败（确认Bug存在）
3. 修复代码
4. 确保测试通过
5. 添加相关边界测试
6. 重构改善代码
```

### 场景3: 重构现有代码
```
用户: "使用TDD重构缓存清理逻辑"

流程:
1. 为现有功能编写测试（如果还没有）
2. 确保测试覆盖所有行为
3. 开始重构
4. 每次修改后运行测试
5. 逐步改善代码结构
```

## TDD 最佳实践

### ✅ 应该做的
- 始终保持测试先行
- 编写可读性强的测试
- 使用表格驱动测试减少重复
- 测试行为而非实现细节
- 保持测试运行快速
- 及时重构测试代码
- 使用有意义的测试名称

### ❌ 不应该做的
- 不要先写实现再补测试
- 不要测试私有方法（测试公共API）
- 不要在测试中使用随机值
- 不要跳过Red阶段
- 不要在Green阶段重构
- 不要编写不稳定的测试
- 不要忽略失败的测试

## 与Git版本管理集成

### TDD 提交流程
```bash
# 1. 完成一个TDD循环后
git add .
git commit -m "feat: 实现缓存TTL功能

- 添加带过期时间的Set方法
- 实现过期检查逻辑
- 包含3个测试用例"

# 2. 使用版本管理技能
# 触发自动版本号更新和tag创建
```

### 提交信息规范
```
feat: 新功能（触发MINOR版本+1）
fix: Bug修复（触发PATCH版本+1）
refactor: 重构（可选PATCH版本+1）
test: 添加测试（通常不触发版本更新）
```

## 工具推荐

### Go 测试工具
```bash
# 运行测试
go test ./...

# 运行测试并显示覆盖度
go test -cover ./...

# 运行特定测试
go test -run TestCache_SetAndGet

# 生成测试覆盖报告
go test -coverprofile=coverage.out
go tool cover -html=coverage.out

# 持续测试（文件变更时自动运行）
go run github.com/cespare/reflex@latest
```

### 辅助工具
- **testify**: 断言库，简化测试代码
- **gomock**: Mock框架
- **goconvey**: BDD风格测试框架
- **gotests**: 自动生成测试函数

## 注意事项

1. **不要过度测试**：测试应该有实际价值
2. **保持测试独立**：测试之间不应该相互依赖
3. **测试要快速**：整个测试套件应该在几秒内完成
4. **避免脆弱测试**：测试不应因无关改动而失败
5. **定期清理**：删除无用测试，保持测试代码整洁
6. **文档化**：测试就是最好的文档

## 示例对话

### 示例1: 开始TDD开发
```
用户: "使用TDD实现一个带LRU淘汰策略的缓存"

AI助手:
好的！我将使用TDD方式实现LRU缓存。

📋 测试计划：
1. TestLRUCache_BasicSetAndGet - 基本的设置和获取
2. TestLRUCache_CapacityLimit - 容量限制测试
3. TestLRUCache_EvictionPolicy - LRU淘汰策略测试
4. TestLRUCache_RecentAccess - 最近访问更新测试

让我们从第一个测试开始...

🔴 Red: 编写测试 TestLRUCache_BasicSetAndGet
[编写测试代码]
运行测试... 测试失败 ✅（符合预期）

🟢 Green: 实现基础功能
[实现代码]
运行测试... 测试通过 ✅

🔵 Refactor: 优化代码结构
[重构代码]
运行测试... 测试通过 ✅

继续下一个测试...
```

### 示例2: Bug修复TDD
```
用户: "缓存并发访问有问题，用TDD修复"

AI助手:
好的！我将使用TDD修复并发问题。

🔴 Red: 编写并发测试
[编写并发测试，使用-race标志]
运行测试... 检测到竞态条件 ✅（确认Bug）

🟢 Green: 修复并发问题
[添加互斥锁或改用sync.Map]
运行测试... 测试通过 ✅

🔵 Refactor: 改善并发实现
[优化锁粒度或改用读写锁]
运行测试... 测试通过 ✅
```
