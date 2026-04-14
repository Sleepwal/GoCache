package cache

import (
	"testing"
	"time"
)

// TestStringCache_BasicSetAndGet 测试基本的设置和获取
func TestStringCache_BasicSetAndGet(t *testing.T) {
	sc := NewStringCache(New())

	sc.Set("key", "hello", 0)

	val, found := sc.Get("key")
	if !found {
		t.Error("expected key to be found")
	}
	if val != "hello" {
		t.Errorf("expected 'hello', got '%s'", val)
	}
}

// TestStringCache_GetNotFound 测试获取不存在的键
func TestStringCache_GetNotFound(t *testing.T) {
	sc := NewStringCache(New())

	val, found := sc.Get("nonexistent")
	if found {
		t.Errorf("expected not found, got '%s'", val)
	}
}

// TestStringCache_Append 测试字符串追加
func TestStringCache_Append(t *testing.T) {
	sc := NewStringCache(New())

	// 追加到新键
	length := sc.Append("key", "hello")
	if length != 5 {
		t.Errorf("expected length 5, got %d", length)
	}

	val, found := sc.Get("key")
	if !found || val != "hello" {
		t.Errorf("expected 'hello', got '%s'", val)
	}

	// 追加到已存在键
	length = sc.Append("key", " world")
	if length != 11 {
		t.Errorf("expected length 11, got %d", length)
	}

	val, found = sc.Get("key")
	if !found || val != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", val)
	}
}

// TestStringCache_Incr 测试自增
func TestStringCache_Incr(t *testing.T) {
	sc := NewStringCache(New())

	// 自增不存在的键
	val, err := sc.Incr("counter")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != 1 {
		t.Errorf("expected 1, got %d", val)
	}

	// 再次自增
	val, err = sc.Incr("counter")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != 2 {
		t.Errorf("expected 2, got %d", val)
	}
}

// TestStringCache_IncrBy 测试指定增量自增
func TestStringCache_IncrBy(t *testing.T) {
	sc := NewStringCache(New())

	val, err := sc.IncrBy("counter", 5)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != 5 {
		t.Errorf("expected 5, got %d", val)
	}

	val, err = sc.IncrBy("counter", 3)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != 8 {
		t.Errorf("expected 8, got %d", val)
	}
}

// TestStringCache_Incr_StringValue 测试字符串值自增
func TestStringCache_Incr_StringValue(t *testing.T) {
	sc := NewStringCache(New())

	sc.Set("counter", "10", 0)

	val, err := sc.Incr("counter")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != 11 {
		t.Errorf("expected 11, got %d", val)
	}
}

// TestStringCache_Incr_NonInteger 测试非整数值自增错误
func TestStringCache_Incr_NonInteger(t *testing.T) {
	sc := NewStringCache(New())

	sc.Set("counter", "not_a_number", 0)

	_, err := sc.Incr("counter")
	if err == nil {
		t.Error("expected error for non-integer value")
	}
}

// TestStringCache_Decr 测试自减
func TestStringCache_Decr(t *testing.T) {
	sc := NewStringCache(New())

	sc.Set("counter", "10", 0)

	val, err := sc.Decr("counter")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != 9 {
		t.Errorf("expected 9, got %d", val)
	}
}

// TestStringCache_DecrBy 测试指定减量自减
func TestStringCache_DecrBy(t *testing.T) {
	sc := NewStringCache(New())

	sc.Set("counter", "10", 0)

	val, err := sc.DecrBy("counter", 3)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != 7 {
		t.Errorf("expected 7, got %d", val)
	}
}

// TestStringCache_GetRange 测试子字符串获取
func TestStringCache_GetRange(t *testing.T) {
	sc := NewStringCache(New())

	sc.Set("key", "hello world", 0)

	// 正常范围 (包含两端)
	val, found := sc.GetRange("key", 0, 4)
	if !found || val != "hello" {
		t.Errorf("expected 'hello', got '%s'", val)
	}

	// 负数索引
	val, found = sc.GetRange("key", -5, -1)
	if !found || val != "world" {
		t.Errorf("expected 'world', got '%s'", val)
	}

	// 超出范围
	val, found = sc.GetRange("key", 0, 100)
	if !found || val != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", val)
	}
}

// TestStringCache_GetRange_NotFound 测试不存在的键
func TestStringCache_GetRange_NotFound(t *testing.T) {
	sc := NewStringCache(New())

	_, found := sc.GetRange("nonexistent", 0, 4)
	if found {
		t.Error("expected not found")
	}
}

// TestStringCache_StrLen 测试字符串长度
func TestStringCache_StrLen(t *testing.T) {
	sc := NewStringCache(New())

	sc.Set("key", "hello", 0)

	length, found := sc.StrLen("key")
	if !found || length != 5 {
		t.Errorf("expected length 5, got %d", length)
	}

	// 中文字符
	sc.Set("cn", "你好世界", 0)
	length, found = sc.StrLen("cn")
	if !found || length != 4 {
		t.Errorf("expected length 4, got %d", length)
	}
}

// TestStringCache_StrLen_NotFound 测试不存在的键长度
func TestStringCache_StrLen_NotFound(t *testing.T) {
	sc := NewStringCache(New())

	_, found := sc.StrLen("nonexistent")
	if found {
		t.Error("expected not found")
	}
}

// TestStringCache_SetRange 测试字符串覆盖
func TestStringCache_SetRange(t *testing.T) {
	sc := NewStringCache(New())

	sc.Set("key", "hello world", 0)

	length := sc.SetRange("key", 6, "redis")
	if length != 11 {
		t.Errorf("expected length 11, got %d", length)
	}

	val, found := sc.Get("key")
	if !found || val != "hello redis" {
		t.Errorf("expected 'hello redis', got '%s'", val)
	}
}

// TestStringCache_GetSet 测试获取并设置新值
func TestStringCache_GetSet(t *testing.T) {
	sc := NewStringCache(New())

	sc.Set("key", "old", 0)

	oldVal, found := sc.GetSet("key", "new")
	if !found || oldVal != "old" {
		t.Errorf("expected old value 'old', got '%s'", oldVal)
	}

	val, found := sc.Get("key")
	if !found || val != "new" {
		t.Errorf("expected 'new', got '%s'", val)
	}
}

// TestStringCache_GetSet_NotFound 测试不存在的键
func TestStringCache_GetSet_NotFound(t *testing.T) {
	sc := NewStringCache(New())

	oldVal, found := sc.GetSet("nonexistent", "value")
	if found {
		t.Errorf("expected not found, got '%s'", oldVal)
	}
}

// TestStringCache_TTL 测试 TTL 过期
func TestStringCache_TTL(t *testing.T) {
	sc := NewStringCache(New())

	sc.Set("key", "value", 50*time.Millisecond)

	// 立即获取应该成功
	val, found := sc.Get("key")
	if !found || val != "value" {
		t.Errorf("expected 'value', got '%s'", val)
	}

	// 等待过期
	time.Sleep(100 * time.Millisecond)

	_, found = sc.Get("key")
	if found {
		t.Error("expected key to be expired")
	}
}

// TestStringCache_IntType 测试整数类型
func TestStringCache_IntType(t *testing.T) {
	sc := NewStringCache(New())

	sc.cache.Set("counter", 42, 0)

	val, err := sc.Incr("counter")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != 43 {
		t.Errorf("expected 43, got %d", val)
	}
}
