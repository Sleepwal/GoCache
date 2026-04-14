package cache

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// StringCache String 类型缓存操作
type StringCache struct {
	cache *MemoryCache
}

// NewStringCache 创建 String 类型缓存
func NewStringCache(cache *MemoryCache) *StringCache {
	if cache == nil {
		cache = New()
	}
	return &StringCache{
		cache: cache,
	}
}

// Set 设置字符串值
func (sc *StringCache) Set(key, value string, ttl time.Duration) {
	sc.cache.Set(key, value, ttl)
}

// Get 获取字符串值
func (sc *StringCache) Get(key string) (string, bool) {
	val, found := sc.cache.Get(key)
	if !found {
		return "", false
	}

	strVal, ok := val.(string)
	if !ok {
		return "", false
	}

	return strVal, true
}

// Append 追加字符串到值末尾
// 如果键不存在，则创建新键
func (sc *StringCache) Append(key, value string) int {
	sc.cache.mu.Lock()
	defer sc.cache.mu.Unlock()

	item, found := sc.cache.items[key]
	var original string

	if found && !item.IsExpired() {
		if strVal, ok := item.Value.(string); ok {
			original = strVal
		}
	}

	newValue := original + value
	sc.cache.items[key] = &Item{
		Value:      newValue,
		Expiration: 0,
	}

	return len(newValue)
}

// Incr 将键的值增加 1
// 如果键不存在，则初始化为 0 后增加 1
// 如果值不是整数，返回错误
func (sc *StringCache) Incr(key string) (int64, error) {
	return sc.IncrBy(key, 1)
}

// IncrBy 将键的值增加指定整数
func (sc *StringCache) IncrBy(key string, n int64) (int64, error) {
	sc.cache.mu.Lock()
	defer sc.cache.mu.Unlock()

	var current int64

	item, found := sc.cache.items[key]
	if found && !item.IsExpired() {
		switch v := item.Value.(type) {
		case string:
			val, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("value is not an integer")
			}
			current = val
		case int:
			current = int64(v)
		case int64:
			current = v
		default:
			return 0, fmt.Errorf("value is not an integer")
		}
	}

	newValue := current + n
	sc.cache.items[key] = &Item{
		Value:      newValue,
		Expiration: 0,
	}

	return newValue, nil
}

// Decr 将键的值减少 1
func (sc *StringCache) Decr(key string) (int64, error) {
	return sc.IncrBy(key, -1)
}

// DecrBy 将键的值减少指定整数
func (sc *StringCache) DecrBy(key string, n int64) (int64, error) {
	return sc.IncrBy(key, -n)
}

// GetRange 获取子字符串
// start 和 end 都支持负数索引（-1 表示最后一个字符）
func (sc *StringCache) GetRange(key string, start, end int) (string, bool) {
	sc.cache.mu.RLock()
	defer sc.cache.mu.RUnlock()

	item, found := sc.cache.items[key]
	if !found || item.IsExpired() {
		return "", false
	}

	strVal, ok := item.Value.(string)
	if !ok {
		return "", false
	}

	runes := []rune(strVal)
	length := len(runes)

	// 处理负数索引
	if start < 0 {
		start = length + start
		if start < 0 {
			start = 0
		}
	}

	if end < 0 {
		end = length + end
	}

	// 边界检查
	if start > length {
		return "", true
	}

	if end >= length {
		end = length - 1
	}

	if start > end {
		return "", true
	}

	if end < 0 {
		return "", true
	}

	return string(runes[start : end+1]), true
}

// StrLen 获取字符串长度
func (sc *StringCache) StrLen(key string) (int, bool) {
	sc.cache.mu.RLock()
	defer sc.cache.mu.RUnlock()

	item, found := sc.cache.items[key]
	if !found || item.IsExpired() {
		return 0, false
	}

	strVal, ok := item.Value.(string)
	if !ok {
		return 0, false
	}

	return len([]rune(strVal)), true
}

// SetRange 覆盖字符串的指定位置
func (sc *StringCache) SetRange(key string, offset int, value string) int {
	sc.cache.mu.Lock()
	defer sc.cache.mu.Unlock()

	item, found := sc.cache.items[key]
	var original string

	if found && !item.IsExpired() {
		if strVal, ok := item.Value.(string); ok {
			original = strVal
		}
	}

	// 如果 offset 超出原始长度，用空字符填充
	runes := []rune(original)
	if offset >= len(runes) {
		padding := strings.Repeat(" ", offset-len(runes))
		original = original + padding
		runes = []rune(original)
	}

	// 替换指定位置的值
	valueRunes := []rune(value)
	for i, r := range valueRunes {
		if offset+i < len(runes) {
			runes[offset+i] = r
		} else {
			runes = append(runes, r)
		}
	}

	newValue := string(runes)
	sc.cache.items[key] = &Item{
		Value:      newValue,
		Expiration: 0,
	}

	return len(runes)
}

// GetSet 设置新值并返回旧值
func (sc *StringCache) GetSet(key, value string) (string, bool) {
	sc.cache.mu.Lock()
	defer sc.cache.mu.Unlock()

	var oldValue string
	var found bool

	item, exists := sc.cache.items[key]
	if exists && !item.IsExpired() {
		if strVal, ok := item.Value.(string); ok {
			oldValue = strVal
			found = true
		}
	}

	sc.cache.items[key] = &Item{
		Value:      value,
		Expiration: 0,
	}

	return oldValue, found
}
