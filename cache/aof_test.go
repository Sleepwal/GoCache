package cache

import (
	"os"
	"testing"
	"time"
)

// TestAOFLogger_LogAndReplay 测试 AOF 日志记录和重放
func TestAOFLogger_LogAndReplay(t *testing.T) {
	aof, err := NewAOFLogger("/tmp/test_aof.log")
	if err != nil {
		t.Fatalf("failed to create AOF logger: %v", err)
	}
	defer os.Remove("/tmp/test_aof.log")
	defer aof.Close()

	// 记录操作（使用新方法保留类型）
	aof.LogSet("key1", "value1", 0)
	aof.LogSet("key2", "value2", 0)
	aof.LogSet("key3", "value3", 0)

	// 创建新缓存并重放
	c := New()
	err = aof.Replay(c)
	if err != nil {
		t.Fatalf("failed to replay AOF: %v", err)
	}

	// 验证数据
	val, found := c.Get("key1")
	if !found || val != "value1" {
		t.Errorf("expected 'value1', got %v", val)
	}

	val, found = c.Get("key2")
	if !found || val != "value2" {
		t.Errorf("expected 'value2', got %v", val)
	}
}

// TestAOFLogger_TypePreservation 测试 AOF 保留 Go 原始类型
func TestAOFLogger_TypePreservation(t *testing.T) {
	aof, err := NewAOFLogger("/tmp/test_aof_types.log")
	if err != nil {
		t.Fatalf("failed to create AOF logger: %v", err)
	}
	defer os.Remove("/tmp/test_aof_types.log")
	defer aof.Close()

	// 记录不同类型的值
	// 注意：gob 对 map 和 slice 的类型保留有限制，只测试基本类型
	aof.LogSet("string_key", "hello", 0)
	aof.LogSet("int_key", int64(42), 0)
	aof.LogSet("bool_key", true, 0)
	aof.LogSet("float_key", 3.14, 0)

	// 重放
	c := New()
	err = aof.Replay(c)
	if err != nil {
		t.Fatalf("failed to replay AOF: %v", err)
	}

	// 验证类型和值
	val, found := c.Get("string_key")
	if !found || val != "hello" {
		t.Errorf("expected 'hello', got %v", val)
	}

	val, found = c.Get("int_key")
	if !found {
		t.Error("int_key not found")
	} else {
		intVal, ok := val.(int64)
		if !ok || intVal != 42 {
			t.Errorf("expected int64(42), got %T(%v)", val, val)
		}
	}

	val, found = c.Get("bool_key")
	if !found {
		t.Error("bool_key not found")
	} else {
		boolVal, ok := val.(bool)
		if !ok || boolVal != true {
			t.Errorf("expected bool(true), got %T(%v)", val, val)
		}
	}

	val, found = c.Get("float_key")
	if !found {
		t.Error("float_key not found")
	} else {
		floatVal, ok := val.(float64)
		if !ok || floatVal != 3.14 {
			t.Errorf("expected float64(3.14), got %T(%v)", val, val)
		}
	}
}

// TestAOFLogger_Close 测试关闭
func TestAOFLogger_Close(t *testing.T) {
	aof, err := NewAOFLogger("/tmp/test_aof_close.log")
	if err != nil {
		t.Fatalf("failed to create AOF logger: %v", err)
	}
	defer os.Remove("/tmp/test_aof_close.log")

	aof.LogSet("key", "value", 0)

	err = aof.Close()
	if err != nil {
		t.Errorf("failed to close AOF logger: %v", err)
	}

	// 关闭后不应再写入
	err = aof.LogSet("key2", "value2", 0)
	if err != nil {
		t.Errorf("expected no error after close, got %v", err)
	}
}

// TestAOFLogger_Rewrite 测试 AOF 重写
func TestAOFLogger_Rewrite(t *testing.T) {
	aof, err := NewAOFLogger("/tmp/test_aof_rewrite.log")
	if err != nil {
		t.Fatalf("failed to create AOF logger: %v", err)
	}
	defer os.Remove("/tmp/test_aof_rewrite.log")
	defer aof.Close()

	// 创建缓存并写入数据（包含不同类型）
	c := New()
	c.Set("key1", "value1", 0)
	c.Set("key2", int64(123), 0)
	c.Set("key3", true, 0)

	// 重写 AOF
	err = aof.Rewrite(c)
	if err != nil {
		t.Fatalf("failed to rewrite AOF: %v", err)
	}

	// 验证重写后的文件可以正确重放且保留类型
	c2 := New()
	err = aof.Replay(c2)
	if err != nil {
		t.Fatalf("failed to replay after rewrite: %v", err)
	}

	if c2.Count() != 3 {
		t.Errorf("expected 3 items after rewrite, got %d", c2.Count())
	}

	// 验证类型保留
	val, _ := c2.Get("key2")
	if _, ok := val.(int64); !ok {
		t.Errorf("expected int64 type after rewrite, got %T", val)
	}
}

// TestAOFLogger_WithTTL 测试带 TTL 的 AOF
func TestAOFLogger_WithTTL(t *testing.T) {
	aof, err := NewAOFLogger("/tmp/test_aof_ttl.log")
	if err != nil {
		t.Fatalf("failed to create AOF logger: %v", err)
	}
	defer os.Remove("/tmp/test_aof_ttl.log")
	defer aof.Close()

	// 记录带过期时间的 SET（0 表示永不过期）
	aof.LogSet("key1", "value1", 0)

	// 重放
	c := New()
	err = aof.Replay(c)
	if err != nil {
		t.Fatalf("failed to replay AOF with TTL: %v", err)
	}

	// key1 应该存在
	_, found := c.Get("key1")
	if !found {
		t.Error("expected 'key1' to exist")
	}
}

// TestAOFLogger_Delete 测试删除操作
func TestAOFLogger_Delete(t *testing.T) {
	aof, err := NewAOFLogger("/tmp/test_aof_delete.log")
	if err != nil {
		t.Fatalf("failed to create AOF logger: %v", err)
	}
	defer os.Remove("/tmp/test_aof_delete.log")
	defer aof.Close()

	// 记录 SET 和 DELETE
	aof.LogSet("key1", "value1", 0)
	aof.LogDelete("key1")

	// 重放
	c := New()
	err = aof.Replay(c)
	if err != nil {
		t.Fatalf("failed to replay AOF: %v", err)
	}

	// key1 应该被删除
	_, found := c.Get("key1")
	if found {
		t.Error("expected 'key1' to be deleted")
	}
}

// TestAOFLogger_EmptyFile 测试空文件重放
func TestAOFLogger_EmptyFile(t *testing.T) {
	aof, err := NewAOFLogger("/tmp/test_aof_empty.log")
	if err != nil {
		t.Fatalf("failed to create AOF logger: %v", err)
	}
	defer os.Remove("/tmp/test_aof_empty.log")
	defer aof.Close()

	c := New()
	err = aof.Replay(c)
	if err != nil {
		t.Errorf("failed to replay empty AOF: %v", err)
	}

	if c.Count() != 0 {
		t.Errorf("expected 0 items, got %d", c.Count())
	}
}

// TestAOFLogger_ExpiredTTL 测试过期 TTL 条目
func TestAOFLogger_ExpiredTTL(t *testing.T) {
	aof, err := NewAOFLogger("/tmp/test_aof_expired.log")
	if err != nil {
		t.Fatalf("failed to create AOF logger: %v", err)
	}
	defer os.Remove("/tmp/test_aof_expired.log")
	defer aof.Close()

	// 记录一个已过期的条目（过期时间设为过去）
	pastExpiration := time.Now().Add(-1 * time.Second).UnixNano()
	aof.LogSet("expired_key", "old_value", pastExpiration)

	// 重放（已过期的条目应该被跳过）
	c := New()
	err = aof.Replay(c)
	if err != nil {
		t.Fatalf("failed to replay AOF: %v", err)
	}

	// 过期的 key 不应该存在
	_, found := c.Get("expired_key")
	if found {
		t.Error("expected expired key to not exist")
	}

	if c.Count() != 0 {
		t.Errorf("expected 0 items, got %d", c.Count())
	}
}
