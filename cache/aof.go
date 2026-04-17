package cache

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"GoCache/logger"
)

// AOFLogger Append-Only File 持久化
type AOFLogger struct {
	file             *os.File
	writer           *bufio.Writer
	mu               sync.Mutex
	enabled          bool
	path             string
	fsyncStrategy    FsyncStrategy
	lastFsync        time.Time
	rewriteSize      int64
	baseSize         int64
	rewriteThreshold float64
	autoRewriteStop  func()
	cache            *MemoryCache
}

// NewAOFLogger 创建 AOF 日志器
func NewAOFLogger(path string) (*AOFLogger, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open AOF file: %w", err)
	}

	logger.Info("AOF logger opened", "path", path)

	return &AOFLogger{
		file:             file,
		writer:           bufio.NewWriter(file),
		enabled:          true,
		path:             path,
		fsyncStrategy:    FsyncEverySec,
		lastFsync:        time.Now(),
		rewriteThreshold: 2.0,
	}, nil
}

func NewAOFLoggerWithConfig(path string, fsyncStrategy FsyncStrategy, rewriteThreshold float64, c *MemoryCache) (*AOFLogger, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open AOF file: %w", err)
	}

	stat, _ := file.Stat()
	baseSize := stat.Size()

	logger.Info("AOF logger opened with config", "path", path, "fsync", fsyncStrategy, "rewrite_threshold", rewriteThreshold, "base_size", baseSize)

	return &AOFLogger{
		file:             file,
		writer:           bufio.NewWriter(file),
		enabled:          true,
		path:             path,
		fsyncStrategy:    fsyncStrategy,
		lastFsync:        time.Now(),
		rewriteSize:      baseSize,
		baseSize:         baseSize,
		rewriteThreshold: rewriteThreshold,
		cache:            c,
	}, nil
}

func (a *AOFLogger) StartAutoRewrite(interval time.Duration) func() {
	if a.cache == nil {
		logger.Warn("AOF auto-rewrite requires cache reference")
		return func() {}
	}

	stopCh := make(chan struct{})
	logger.Info("AOF auto-rewrite monitor started", "interval", interval.String())

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				a.checkAutoRewrite()
			case <-stopCh:
				logger.Info("AOF auto-rewrite monitor stopped")
				return
			}
		}
	}()

	a.autoRewriteStop = func() { close(stopCh) }
	return a.autoRewriteStop
}

func (a *AOFLogger) checkAutoRewrite() {
	if a.rewriteThreshold <= 0 || a.baseSize <= 0 {
		return
	}

	a.mu.Lock()
	currentSize := a.rewriteSize
	a.mu.Unlock()

	ratio := float64(currentSize) / float64(a.baseSize)
	if ratio >= a.rewriteThreshold {
		logger.Info("AOF auto-rewrite triggered", "current_size", currentSize, "base_size", a.baseSize, "ratio", ratio)
		if a.cache != nil {
			if err := a.Rewrite(a.cache); err != nil {
				logger.Error("AOF auto-rewrite failed", "error", err)
			}
		}
	}
}

// encodedItem gob 包装器，用于类型保留
type encodedItem struct {
	Value any
}

// encodeValue 使用 gob 编码值，然后 base64 编码为安全字符串
func encodeValue(value any) (string, error) {
	item := encodedItem{Value: value}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(item); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// decodeValue 解码 base64 + gob 还原 Go 原始类型
func decodeValue(encoded string) (any, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewReader(data)
	dec := gob.NewDecoder(buf)

	var item encodedItem
	if err := dec.Decode(&item); err != nil {
		return nil, err
	}
	return item.Value, nil
}

// Log 记录操作到 AOF 文件
func (a *AOFLogger) Log(command string, args ...string) error {
	if !a.enabled {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	timestamp := time.Now().UnixNano()
	line := fmt.Sprintf("%d %s %s\n", timestamp, command, strings.Join(args, " "))

	_, err := a.writer.WriteString(line)
	if err != nil {
		logger.Error("AOF write failed", "command", command, "error", err)
		return fmt.Errorf("failed to write AOF log: %w", err)
	}

	a.rewriteSize += int64(len(line))

	switch a.fsyncStrategy {
	case FsyncAlways:
		if err := a.writer.Flush(); err != nil {
			return err
		}
		if err := a.file.Sync(); err != nil {
			logger.Error("AOF fsync failed", "error", err)
			return err
		}
	case FsyncEverySec:
		if time.Since(a.lastFsync) >= time.Second {
			if err := a.writer.Flush(); err != nil {
				return err
			}
			if err := a.file.Sync(); err != nil {
				logger.Error("AOF fsync failed", "error", err)
			}
			a.lastFsync = time.Now()
		}
	case FsyncNone:
	}

	return nil
}

// LogSet 记录 SET 操作，值使用 gob+base64 编码保留类型
func (a *AOFLogger) LogSet(key string, value any, expiration int64) error {
	if !a.enabled {
		return nil
	}

	encoded, err := encodeValue(value)
	if err != nil {
		return fmt.Errorf("failed to encode value: %w", err)
	}

	return a.Log("SET", key, encoded, fmt.Sprintf("%d", expiration))
}

// LogDelete 记录 DELETE 操作
func (a *AOFLogger) LogDelete(key string) error {
	return a.Log("DELETE", key)
}

// Close 关闭 AOF 文件
func (a *AOFLogger) Close() error {
	a.enabled = false
	logger.Info("AOF logger closing", "path", a.path)
	if err := a.writer.Flush(); err != nil {
		logger.Error("AOF flush on close failed", "error", err)
		return err
	}
	return a.file.Close()
}

func (a *AOFLogger) Flush() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.writer.Flush()
}

// Rewrite 重写 AOF 文件（压缩大小）
func (a *AOFLogger) Rewrite(cache *MemoryCache) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	logger.Info("AOF rewrite started", "path", a.path)

	tmpPath := a.path + ".tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		logger.Error("AOF rewrite: failed to create temp file", "path", tmpPath, "error", err)
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	cache.mu.RLock()
	defer cache.mu.RUnlock()

	writer := bufio.NewWriter(tmpFile)
	timestamp := time.Now().UnixNano()
	written := 0

	for key, item := range cache.items {
		encoded, err := encodeValue(item.Value)
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			logger.Error("AOF rewrite: failed to encode value", "key", key, "error", err)
			return fmt.Errorf("failed to encode value for key %s: %w", key, err)
		}
		line := fmt.Sprintf("%d SET %s %s %d\n", timestamp, key, encoded, item.Expiration)
		if _, err := writer.WriteString(line); err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			logger.Error("AOF rewrite: failed to write line", "error", err)
			return fmt.Errorf("failed to write AOF: %w", err)
		}
		written++
	}

	if err := writer.Flush(); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}

	tmpFile.Close()

	if err := a.writer.Flush(); err != nil {
		return err
	}
	if err := a.file.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, a.path); err != nil {
		logger.Error("AOF rewrite: failed to rename temp file", "error", err)
		a.file, _ = os.OpenFile(a.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		a.writer = bufio.NewWriter(a.file)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	a.file, err = os.OpenFile(a.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Error("AOF rewrite: failed to reopen file", "error", err)
		return fmt.Errorf("failed to reopen AOF file: %w", err)
	}
	a.writer = bufio.NewWriter(a.file)
	a.rewriteSize = 0
	a.baseSize = 0

	logger.Info("AOF rewrite completed", "keys_written", written, "path", a.path)
	return nil
}

// Replay 重放 AOF 文件到缓存
func (a *AOFLogger) Replay(cache *MemoryCache) error {
	logger.Info("AOF replay started", "path", a.path)

	file, err := os.Open(a.path)
	if err != nil {
		logger.Error("AOF replay: failed to open file", "path", a.path, "error", err)
		return fmt.Errorf("failed to open AOF file: %w", err)
	}
	defer file.Close()

	applied := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if err := a.parseAndApply(cache, line); err != nil {
			logger.Warn("AOF replay: failed to parse line", "line", line, "error", err)
			return fmt.Errorf("failed to parse AOF line: %w", err)
		}
		applied++
	}

	if err := scanner.Err(); err != nil {
		logger.Error("AOF replay: scanner error", "error", err)
		return err
	}

	logger.Info("AOF replay completed", "lines_applied", applied, "path", a.path)
	return nil
}

// parseAndApply 解析并应用 AOF 日志行
func (a *AOFLogger) parseAndApply(cache *MemoryCache, line string) error {
	// 格式: TIMESTAMP COMMAND key encoded_value expiration
	parts := strings.SplitN(line, " ", 5)
	if len(parts) < 3 {
		return nil // 跳过无效行
	}

	command := parts[1]

	switch command {
	case "SET":
		if len(parts) < 5 {
			return nil
		}
		key := parts[2]
		encodedValue := parts[3]

		// 解码值（保留原始 Go 类型）
		value, err := decodeValue(encodedValue)
		if err != nil {
			return fmt.Errorf("failed to decode value for key %s: %w", key, err)
		}

		// 解析过期时间
		var expiration int64
		fmt.Sscanf(parts[4], "%d", &expiration)

		// 如果已过期，跳过
		if expiration > 0 {
			ttl := time.Until(time.Unix(0, expiration))
			if ttl < 0 {
				return nil // 已过期，跳过
			}
			cache.Set(key, value, ttl)
		} else {
			cache.Set(key, value, 0)
		}

	case "DELETE":
		if len(parts) >= 3 {
			cache.Delete(parts[2])
		}
	}

	return nil
}
