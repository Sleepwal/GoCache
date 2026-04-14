package cache

import (
	"testing"
	"time"
)

// TestHashCache_HSet 测试设置字段
func TestHashCache_HSet(t *testing.T) {
	hc := NewHashCache()

	fields := map[string]any{
		"name": "Alice",
		"age":  30,
		"city": "Beijing",
	}

	count := hc.HSet("user1", 0, fields)
	if count != 3 {
		t.Errorf("expected 3 new fields, got %d", count)
	}

	// 更新已存在字段
	fields2 := map[string]any{
		"age":  31,
		"email": "alice@example.com",
	}
	count = hc.HSet("user1", 0, fields2)
	if count != 1 {
		t.Errorf("expected 1 new field, got %d", count)
	}
}

// TestHashCache_HSetSingle 测试设置单个字段
func TestHashCache_HSetSingle(t *testing.T) {
	hc := NewHashCache()

	newField := hc.HSetSingle("user1", "name", 0, "Bob")
	if !newField {
		t.Error("expected new field")
	}

	updated := hc.HSetSingle("user1", "name", 0, "Alice")
	if updated {
		t.Error("expected field to be updated, not new")
	}
}

// TestHashCache_HGet 测试获取字段值
func TestHashCache_HGet(t *testing.T) {
	hc := NewHashCache()

	hc.HSetSingle("user1", "name", 0, "Alice")

	val, found := hc.HGet("user1", "name")
	if !found || val != "Alice" {
		t.Errorf("expected 'Alice', got %v", val)
	}

	_, found = hc.HGet("user1", "nonexistent")
	if found {
		t.Error("expected not found")
	}
}

// TestHashCache_HGetAll 测试获取所有字段
func TestHashCache_HGetAll(t *testing.T) {
	hc := NewHashCache()

	hc.HSet("user1", 0, map[string]any{
		"name": "Alice",
		"age":  30,
	})

	fields, found := hc.HGetAll("user1")
	if !found || len(fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(fields))
	}
	if fields["name"] != "Alice" || fields["age"] != 30 {
		t.Errorf("unexpected field values: %v", fields)
	}
}

// TestHashCache_HDel 测试删除字段
func TestHashCache_HDel(t *testing.T) {
	hc := NewHashCache()

	hc.HSet("user1", 0, map[string]any{
		"name": "Alice",
		"age":  30,
		"city": "Beijing",
	})

	deleted := hc.HDel("user1", "name", "age")
	if deleted != 2 {
		t.Errorf("expected 2 deleted, got %d", deleted)
	}

	length, _ := hc.HLen("user1")
	if length != 1 {
		t.Errorf("expected 1 field, got %d", length)
	}
}

// TestHashCache_HExists 测试字段是否存在
func TestHashCache_HExists(t *testing.T) {
	hc := NewHashCache()

	hc.HSetSingle("user1", "name", 0, "Alice")

	if !hc.HExists("user1", "name") {
		t.Error("expected 'name' to exist")
	}

	if hc.HExists("user1", "nonexistent") {
		t.Error("expected 'nonexistent' to not exist")
	}
}

// TestHashCache_HLen 测试字段数量
func TestHashCache_HLen(t *testing.T) {
	hc := NewHashCache()

	hc.HSet("user1", 0, map[string]any{
		"name": "Alice",
		"age":  30,
		"city": "Beijing",
	})

	length, found := hc.HLen("user1")
	if !found || length != 3 {
		t.Errorf("expected 3 fields, got %d", length)
	}
}

// TestHashCache_HKeys 测试获取所有字段名
func TestHashCache_HKeys(t *testing.T) {
	hc := NewHashCache()

	hc.HSet("user1", 0, map[string]any{
		"name": "Alice",
		"age":  30,
	})

	keys, found := hc.HKeys("user1")
	if !found || len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}

	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}

	if !keySet["name"] || !keySet["age"] {
		t.Errorf("unexpected keys: %v", keys)
	}
}

// TestHashCache_HVals 测试获取所有字段值
func TestHashCache_HVals(t *testing.T) {
	hc := NewHashCache()

	hc.HSet("user1", 0, map[string]any{
		"name": "Alice",
		"age":  30,
	})

	vals, found := hc.HVals("user1")
	if !found || len(vals) != 2 {
		t.Errorf("expected 2 values, got %d", len(vals))
	}

	valSet := make(map[any]bool)
	for _, v := range vals {
		valSet[v] = true
	}

	if !valSet["Alice"] || !valSet[30] {
		t.Errorf("unexpected values: %v", vals)
	}
}

// TestHashCache_HSetNX 测试字段不存在时设置
func TestHashCache_HSetNX(t *testing.T) {
	hc := NewHashCache()

	success := hc.HSetNX("user1", "name", 0, "Alice")
	if !success {
		t.Error("expected HSetNX to succeed for new field")
	}

	success = hc.HSetNX("user1", "name", 0, "Bob")
	if success {
		t.Error("expected HSetNX to fail for existing field")
	}

	val, _ := hc.HGet("user1", "name")
	if val != "Alice" {
		t.Errorf("expected 'Alice', got %v", val)
	}
}

// TestHashCache_HIncrBy 测试字段值自增
func TestHashCache_HIncrBy(t *testing.T) {
	hc := NewHashCache()

	hc.HSetSingle("user1", "age", 0, 10)

	val, err := hc.HIncrBy("user1", "age", 0, 5)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != 15 {
		t.Errorf("expected 15, got %d", val)
	}

	// 自增不存在的字段
	val, err = hc.HIncrBy("user1", "score", 0, 100)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != 100 {
		t.Errorf("expected 100, got %d", val)
	}
}

// TestHashCache_Delete 测试删除 Hash
func TestHashCache_Delete(t *testing.T) {
	hc := NewHashCache()

	hc.HSetSingle("user1", "name", 0, "Alice")

	deleted := hc.Delete("user1")
	if !deleted {
		t.Error("expected delete to succeed")
	}

	_, found := hc.HGet("user1", "name")
	if found {
		t.Error("expected hash to be deleted")
	}
}

// TestHashCache_Exists 测试 Hash 是否存在
func TestHashCache_Exists(t *testing.T) {
	hc := NewHashCache()

	hc.HSetSingle("user1", "name", 0, "Alice")

	if !hc.Exists("user1") {
		t.Error("expected hash to exist")
	}

	if hc.Exists("nonexistent") {
		t.Error("expected nonexistent hash to not exist")
	}
}

// TestHashCache_Keys 测试获取所有键
func TestHashCache_Keys(t *testing.T) {
	hc := NewHashCache()

	hc.HSetSingle("user1", "name", 0, "Alice")
	hc.HSetSingle("user2", "name", 0, "Bob")
	hc.HSetSingle("user3", "name", 0, "Charlie")

	keys := hc.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}
}

// TestHashCache_Clear 测试清空
func TestHashCache_Clear(t *testing.T) {
	hc := NewHashCache()

	hc.HSetSingle("user1", "name", 0, "Alice")
	hc.HSetSingle("user2", "name", 0, "Bob")

	hc.Clear()

	if hc.Count() != 0 {
		t.Errorf("expected 0 hashes after clear, got %d", hc.Count())
	}
}

// TestHashCache_TTL 测试 TTL 过期
func TestHashCache_TTL(t *testing.T) {
	hc := NewHashCache()

	hc.HSet("user1", 50*time.Millisecond, map[string]any{
		"name": "Alice",
		"age":  30,
	})

	// 立即获取应该成功
	fields, found := hc.HGetAll("user1")
	if !found || len(fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(fields))
	}

	// 等待过期
	time.Sleep(100 * time.Millisecond)

	_, found = hc.HGetAll("user1")
	if found {
		t.Error("expected hash to be expired")
	}
}

// TestHashCache_HDel_NotFound 测试删除不存在的字段
func TestHashCache_HDel_NotFound(t *testing.T) {
	hc := NewHashCache()

	hc.HSetSingle("user1", "name", 0, "Alice")

	deleted := hc.HDel("user1", "nonexistent")
	if deleted != 0 {
		t.Errorf("expected 0 deleted, got %d", deleted)
	}
}

// TestHashCache_HGetAll_NotFound 测试不存在的 Hash
func TestHashCache_HGetAll_NotFound(t *testing.T) {
	hc := NewHashCache()

	_, found := hc.HGetAll("nonexistent")
	if found {
		t.Error("expected not found")
	}
}
