package cache

import (
	"sync"
	"time"
)

// hashData Hash 数据结构
type hashData struct {
	fields     map[string]any
	expiration int64 // 过期时间戳(纳秒), 0表示永不过期
}

// HashCache Hash 类型缓存操作
type HashCache struct {
	cache map[string]*hashData
	mu    sync.RWMutex
}

// NewHashCache 创建 Hash 类型缓存
func NewHashCache() *HashCache {
	return &HashCache{
		cache: make(map[string]*hashData),
	}
}

// isExpired 检查 Hash 是否过期
func (hd *hashData) isExpired() bool {
	if hd.expiration == 0 {
		return false
	}
	return time.Now().UnixNano() > hd.expiration
}

// getOrCreate 获取或创建 Hash 数据
func (hc *HashCache) getOrCreate(key string, ttl time.Duration) *hashData {
	hd, found := hc.cache[key]
	if !found || hd.isExpired() {
		hd = &hashData{
			fields: make(map[string]any),
		}
		if ttl > 0 {
			hd.expiration = time.Now().Add(ttl).UnixNano()
		}
		hc.cache[key] = hd
	}
	return hd
}

// HSet 设置一个或多个字段值
func (hc *HashCache) HSet(key string, ttl time.Duration, fields map[string]any) int {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hd := hc.getOrCreate(key, ttl)

	count := 0
	for field, value := range fields {
		_, exists := hd.fields[field]
		hd.fields[field] = value
		if !exists {
			count++
		}
	}

	return count
}

// HSetSingle 设置单个字段值
func (hc *HashCache) HSetSingle(key, field string, ttl time.Duration, value any) bool {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hd := hc.getOrCreate(key, ttl)

	_, exists := hd.fields[field]
	hd.fields[field] = value

	return !exists
}

// HGet 获取字段值
func (hc *HashCache) HGet(key, field string) (any, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	hd, found := hc.cache[key]
	if !found || hd.isExpired() {
		return nil, false
	}

	val, exists := hd.fields[field]
	return val, exists
}

// HGetAll 获取所有字段和值
func (hc *HashCache) HGetAll(key string) (map[string]any, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	hd, found := hc.cache[key]
	if !found || hd.isExpired() {
		return nil, false
	}

	// 返回副本
	result := make(map[string]any, len(hd.fields))
	for f, v := range hd.fields {
		result[f] = v
	}

	return result, true
}

// HDel 删除一个或多个字段
func (hc *HashCache) HDel(key string, fields ...string) int {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hd, found := hc.cache[key]
	if !found || hd.isExpired() {
		return 0
	}

	deleted := 0
	for _, field := range fields {
		if _, exists := hd.fields[field]; exists {
			delete(hd.fields, field)
			deleted++
		}
	}

	return deleted
}

// HExists 检查字段是否存在
func (hc *HashCache) HExists(key, field string) bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	hd, found := hc.cache[key]
	if !found || hd.isExpired() {
		return false
	}

	_, exists := hd.fields[field]
	return exists
}

// HLen 获取字段数量
func (hc *HashCache) HLen(key string) (int, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	hd, found := hc.cache[key]
	if !found || hd.isExpired() {
		return 0, false
	}

	return len(hd.fields), true
}

// HKeys 获取所有字段名
func (hc *HashCache) HKeys(key string) ([]string, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	hd, found := hc.cache[key]
	if !found || hd.isExpired() {
		return nil, false
	}

	keys := make([]string, 0, len(hd.fields))
	for field := range hd.fields {
		keys = append(keys, field)
	}

	return keys, true
}

// HVals 获取所有字段值
func (hc *HashCache) HVals(key string) ([]any, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	hd, found := hc.cache[key]
	if !found || hd.isExpired() {
		return nil, false
	}

	vals := make([]any, 0, len(hd.fields))
	for _, value := range hd.fields {
		vals = append(vals, value)
	}

	return vals, true
}

// HSetNX 字段不存在时设置值
func (hc *HashCache) HSetNX(key, field string, ttl time.Duration, value any) bool {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hd := hc.getOrCreate(key, ttl)

	if _, exists := hd.fields[field]; exists {
		return false
	}

	hd.fields[field] = value
	return true
}

// HIncrBy 将字段的值增加指定整数
func (hc *HashCache) HIncrBy(key, field string, ttl time.Duration, n int64) (int64, error) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hd := hc.getOrCreate(key, ttl)

	var current int64
	if val, exists := hd.fields[field]; exists {
		switch v := val.(type) {
		case int:
			current = int64(v)
		case int64:
			current = v
		case string:
			parsed, err := parseIntString(v)
			if err != nil {
				return 0, err
			}
			current = parsed
		default:
			return 0, ErrNotInteger
		}
	}

	newValue := current + n
	hd.fields[field] = newValue

	return newValue, nil
}

// Delete 删除整个 Hash
func (hc *HashCache) Delete(key string) bool {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	_, found := hc.cache[key]
	if !found {
		return false
	}

	delete(hc.cache, key)
	return true
}

// Exists 检查 Hash 是否存在
func (hc *HashCache) Exists(key string) bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	hd, found := hc.cache[key]
	if !found {
		return false
	}

	return !hd.isExpired()
}

// Keys 返回所有未过期的键
func (hc *HashCache) Keys() []string {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	keys := make([]string, 0, len(hc.cache))
	for key, hd := range hc.cache {
		if !hd.isExpired() {
			keys = append(keys, key)
		}
	}

	return keys
}

// Clear 清空所有 Hash
func (hc *HashCache) Clear() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.cache = make(map[string]*hashData)
}

// Count 返回 Hash 数量
func (hc *HashCache) Count() int {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	return len(hc.cache)
}

// ErrNotInteger 非整数错误
var ErrNotInteger = &hashError{"value is not an integer"}

// hashError Hash 操作错误
type hashError struct {
	msg string
}

func (e *hashError) Error() string {
	return e.msg
}

// parseIntString 解析整数字符串
func parseIntString(s string) (int64, error) {
	var result int64
	_, err := sscanf(s, "%d", &result)
	if err != nil {
		return 0, err
	}
	return result, nil
}

// sscanf 简易字符串格式化
func sscanf(s, format string, args ...any) (int, error) {
	switch format {
	case "%d":
		if len(args) == 0 {
			return 0, ErrNotInteger
		}
		if ptr, ok := args[0].(*int64); ok {
			var val int64
			_, err := parseInt64(s, &val)
			if err != nil {
				return 0, err
			}
			*ptr = val
			return 1, nil
		}
	}
	return 0, ErrNotInteger
}

// parseInt64 解析 64 位整数
func parseInt64(s string, result *int64) (int, error) {
	negative := false
	start := 0
	if len(s) > 0 && s[0] == '-' {
		negative = true
		start = 1
	}

	var val int64
	for i := start; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, ErrNotInteger
		}
		val = val*10 + int64(c-'0')
	}

	if negative {
		val = -val
	}
	*result = val
	return 1, nil
}
