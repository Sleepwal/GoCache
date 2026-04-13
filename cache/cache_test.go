package cache

import (
	"sync"
	"testing"
	"time"
)

func TestSetAndGet(t *testing.T) {
	c := New()

	c.Set("key1", "value1", 0)

	value, found := c.Get("key1")
	if !found {
		t.Error("Expected to find key1, but not found")
	}

	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}
}

func TestGetNotFound(t *testing.T) {
	c := New()

	_, found := c.Get("nonexistent")
	if found {
		t.Error("Expected not to find key, but found it")
	}
}

func TestDelete(t *testing.T) {
	c := New()

	c.Set("key1", "value1", 0)
	
	deleted := c.Delete("key1")
	if !deleted {
		t.Error("Expected to delete key1, but failed")
	}

	_, found := c.Get("key1")
	if found {
		t.Error("Expected key1 to be deleted, but still exists")
	}

	deleted = c.Delete("nonexistent")
	if deleted {
		t.Error("Expected delete to return false for nonexistent key")
	}
}

func TestExists(t *testing.T) {
	c := New()

	c.Set("key1", "value1", 0)

	if !c.Exists("key1") {
		t.Error("Expected key1 to exist")
	}

	if c.Exists("nonexistent") {
		t.Error("Expected nonexistent key to not exist")
	}
}

func TestKeys(t *testing.T) {
	c := New()

	c.Set("key1", "value1", 0)
	c.Set("key2", "value2", 0)
	c.Set("key3", "value3", 0)

	keys := c.Keys()
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}
}

func TestClear(t *testing.T) {
	c := New()

	c.Set("key1", "value1", 0)
	c.Set("key2", "value2", 0)

	c.Clear()

	if c.Count() != 0 {
		t.Errorf("Expected 0 items after clear, got %d", c.Count())
	}
}

func TestCount(t *testing.T) {
	c := New()

	if c.Count() != 0 {
		t.Errorf("Expected 0 items initially, got %d", c.Count())
	}

	c.Set("key1", "value1", 0)
	c.Set("key2", "value2", 0)

	if c.Count() != 2 {
		t.Errorf("Expected 2 items, got %d", c.Count())
	}
}

func TestTTLExpiration(t *testing.T) {
	c := New()

	c.Set("key1", "value1", 50*time.Millisecond)

	// Should exist before expiration
	if !c.Exists("key1") {
		t.Error("Expected key1 to exist before expiration")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should not exist after expiration
	if c.Exists("key1") {
		t.Error("Expected key1 to be expired")
	}

	_, found := c.Get("key1")
	if found {
		t.Error("Expected to not get expired key")
	}
}

func TestTTLNeverExpire(t *testing.T) {
	c := New()

	c.Set("key1", "value1", 0)

	time.Sleep(50 * time.Millisecond)

	if !c.Exists("key1") {
		t.Error("Expected key1 to exist (no expiration set)")
	}
}

func TestDeleteExpired(t *testing.T) {
	c := New()

	c.Set("key1", "value1", 50*time.Millisecond)
	c.Set("key2", "value2", 0)

	time.Sleep(100 * time.Millisecond)

	// Manually trigger cleanup
	c.DeleteExpired()

	if c.Count() != 1 {
		t.Errorf("Expected 1 item after cleanup, got %d", c.Count())
	}
}

func TestStartEviction(t *testing.T) {
	c := New()

	c.Set("key1", "value1", 50*time.Millisecond)
	c.Set("key2", "value2", 0)

	// Start eviction with 100ms interval
	stop := c.StartEviction(100 * time.Millisecond)

	// Wait for eviction to run
	time.Sleep(200 * time.Millisecond)

	if c.Count() != 1 {
		t.Errorf("Expected 1 item after eviction, got %d", c.Count())
	}

	// Stop eviction
	stop()
}

func TestConcurrentAccess(t *testing.T) {
	c := New()

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c.Set("key", i, 0)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Get("key")
		}()
	}

	// Concurrent deletes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Delete("key")
		}()
	}

	wg.Wait()
}
