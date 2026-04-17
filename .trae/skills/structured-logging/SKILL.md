---
name: "structured-logging"
description: "为 Go 项目提供生产级结构化日志系统，支持日志级别、调用者信息、异步写入、文件轮转和全局 API。当用户需要添加、改进或标准化 Go 项目的日志记录时调用此技能。"
---

# 结构化日志记录技能

本技能为 Go 项目提供系统化、生产就绪的日志记录机制。确保当系统运行过程中出现错误或异常时，开发人员能够通过日志信息快速、准确地定位到问题发生的具体代码位置、执行上下文及相关参数。

## 快速开始

```go
import "GoCache/logger"

// 在程序启动时初始化
func main() {
    logger.Init(
        logger.WithLevel(logger.INFO),
        logger.WithModule("main"),
    )
    defer logger.Close()

    logger.Info("server started", "port", 8080)
}
```

## 日志级别规范

| 级别 | 常量 | 粒度 | 使用场景 |
|------|------|------|----------|
| DEBUG | `logger.DEBUG` | 细粒度调试信息 | 调试变量值、中间状态、循环迭代 |
| INFO | `logger.INFO` | 关键业务流程节点 | 启动/关闭、配置变更、重要操作完成 |
| WARN | `logger.WARN` | 异常但不阻塞的情况 | 使用已弃用 API、接近容量上限、重试操作 |
| ERROR | `logger.ERROR` | 需要关注的操作失败 | 数据库连接失败、业务规则违反 |
| FATAL | `logger.FATAL` | 导致系统退出的致命错误 | 关键配置缺失、不可恢复的状态 |

## 日志格式规范

每条日志包含以下字段：

```
2006-01-02 15:04:05.000 [级别] [文件.go:行号 函数名] 消息 key1=value1 key2=value2
```

- **时间戳**：`YYYY-MM-DD HH:MM:SS.mmm`（毫秒精度）
- **级别**：DEBUG / INFO / WARN / ERROR / FATAL
- **调用者信息**：`文件名.go:行号 函数名`（自动获取，无需手动传入）
- **消息**：人类可读的描述文本
- **结构化字段**：键值对形式，如 `key=value`

## API 使用

### 全局 API（推荐用于大多数场景）

```go
// 结构化日志
logger.Debug("debug message", "key", value, "another_key", anotherValue)
logger.Info("info message", "key", value)
logger.Warn("warn message", "key", value)
logger.Error("error message", "key", value)
logger.ErrorErr("operation failed", err, "retry", 3)
logger.Fatal("fatal message", "key", value)

// 格式化日志
logger.Infof("server started on port %d", port)
logger.Errorf("failed to connect to %s: %v", host, err)

// 级别控制
logger.SetLevel(logger.DEBUG)
level := logger.GetLevel()

// 确保所有缓存日志已写入
logger.Sync()
```

### 实例化 API（用于上下文感知日志）

```go
// 创建日志实例
l := logger.New(
    logger.WithLevel(logger.INFO),
    logger.WithModule("cache"),
    logger.WithColorize(true),
)

// 创建继承父级字段的子日志
requestLogger := l.With("request_id", req.ID, "user", req.UserID)
requestLogger.Info("processing request")

// 完成后关闭（仅关闭顶级实例，不关闭子日志）
l.Close()
```

### 基于配置的初始化

```go
cfg := logger.Config{
    Level:        "INFO",
    Module:       "server",
    Colorize:     true,
    FileOutput:   true,
    FilePath:     "./logs/app.log",
    MaxSize:      100 * 1024 * 1024, // 单文件最大 100MB
    MaxBackups:   5,                 // 最多保留 5 个备份文件
    MaxAge:       30,                // 日志保留 30 天
    RotateByDate: false,             // 设为 true 则按日期轮转
}

logger.InitFromConfig(cfg)
```

## 日志输出方式

### 控制台输出（默认）
- 终端彩色输出，提高可读性
- 写入文件时自动使用纯文本格式

### 文件输出
- 支持按大小轮转（默认 100MB）
- 支持按日期轮转
- 自动清理超限/超龄的备份文件（可配置最大数量和保留天数）
- 日志目录不存在时自动创建

### 多输出支持
```go
// 添加额外的输出目标
logger.AddWriter(fileWriter)

// 或显式设置输出目标
logger.Init(logger.WithWriters(os.Stdout, fileWriter))
```

## 集成指南

### 1. 替换 `fmt`/`log` 为结构化日志

```go
// 替换前
log.Printf("server started on port %d", port)
fmt.Println("cache hit for key:", key)

// 替换后
logger.Info("server started", "port", port)
logger.Debug("cache hit", "key", key)
```

### 2. 使用合适的日志级别

- **INFO**：系统启动/关闭、主要状态变更、HTTP 请求记录
- **WARN**：重试操作、接近容量限制、使用已弃用功能
- **ERROR**：操作失败、连接错误、业务规则违反
- **DEBUG**：详细变量值、循环迭代、缓存命中/未命中

### 3. 在错误日志中包含上下文信息

```go
// 好的做法
logger.ErrorErr("database query failed", err,
    "table", "users",
    "query_id", qid,
    "duration", elapsed.String(),
)

// 避免的做法
logger.Error("error occurred") // 没有上下文
logger.Error("database query failed") // 没有错误详情
```

### 4. 使用子日志记录请求上下文

```go
func handleRequest(w http.ResponseWriter, r *http.Request) {
    reqLog := logger.With(
        "request_id", generateID(),
        "method", r.Method,
        "path", r.URL.Path,
    )
    reqLog.Info("request received")
    
    // 在处理函数中使用 reqLog
    // ...
}
```

### 5. 在程序退出时关闭日志

```go
func main() {
    defer logger.Close()
    // ...
}
```

## 性能优化

- 日志写入采用异步方式，通过缓冲 channel（容量 4096）处理
- 当 channel 满时，日志降级为同步写入以防止数据丢失
- 在关键操作前使用 `logger.Sync()` 确保日志已刷新
- 使用对象池（`sync.Pool`）复用日志条目对象，减少 GC 压力

## 测试验证

日志系统包含全面的单元测试：

```bash
go test ./logger/... -v -count=1
```

测试覆盖：
- 日志级别输出和过滤
- 输出格式验证（时间戳、调用者信息、消息）
- 结构化字段渲染
- 错误对象日志
- 格式化日志方法
- 子日志字段继承
- 模块字段支持
- 运行时级别变更
- 多输出支持
- 关闭和同步行为
- 按大小文件轮转
- 按最大备份数轮转
- 配置默认值
- 全局 API 功能
- 并发日志安全（10 个协程 x 100 次迭代）
- 调用者信息准确性

## 文件结构

```
logger/
├── logger.go       # 核心引擎：级别定义、Logger 结构体、异步处理、格式化
├── config.go       # 配置系统：Option 函数、Config 结构体、NewFromConfig
├── writer.go       # 文件轮转：RotatingFileWriter 支持按大小/日期轮转
├── global.go       # 全局 API：Init、Info、Error、Sync 等便捷函数
└── logger_test.go  # 22 个单元测试，覆盖全部功能
```
