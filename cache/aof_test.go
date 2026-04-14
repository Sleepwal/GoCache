package cache

import (
	"os"
	"testing"
)

// TestAOFLogger_LogAndReplay 测试 AOF 日志记录和重放
func TestAOFLogger_LogAndReplay(t *testing.T) {
	aof, err := NewAOFLogger("/tmp/test_aof.log")
	if err != nil {
		t.Fatalf("failed to create AOF logger: %v", err)
	}
	defer os.Remove("/tmp/test_aof.log")
	defer aof.Close()

	// 记录操作
	aof.Log("SET", "key1 value1 0")
	aof.Log("SET", "key2 value2 0")
	aof.Log("SET", "key3 value3 0")

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

// TestAOFLogger_Close 测试关闭
func TestAOFLogger_Close(t *testing.T) {
	aof, err := NewAOFLogger("/tmp/test_aof_close.log")
	if err != nil {
		t.Fatalf("failed to create AOF logger: %v", err)
	}
	defer os.Remove("/tmp/test_aof_close.log")

	aof.Log("SET", "key value 0")

	err = aof.Close()
	if err != nil {
		t.Errorf("failed to close AOF logger: %v", err)
	}

	// 关闭后不应再写入
	err = aof.Log("SET", "key2 value2 0")
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

	// 创建缓存并写入数据
	c := New()
	c.Set("key1", "value1", 0)
	c.Set("key2", "value2", 0)
	c.Set("key3", "value3", 0)

	// 重写 AOF
	err = aof.Rewrite(c)
	if err != nil {
		t.Fatalf("failed to rewrite AOF: %v", err)
	}

	// 验证重写后的文件可以正确重放
	c2 := New()
	err = aof.Replay(c2)
	if err != nil {
		t.Fatalf("failed to replay after rewrite: %v", err)
	}

	if c2.Count() != 3 {
		t.Errorf("expected 3 items after rewrite, got %d", c2.Count())
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

	// 记录带过期时间的 SET
	aof.Log("SET", "key1", "value1", "0")

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
	aof.Log("SET", "key1", "value1", "0")
	aof.Log("DELETE", "key1")

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
