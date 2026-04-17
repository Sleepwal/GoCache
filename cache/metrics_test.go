package cache

import (
	"strings"
	"testing"
	"time"
)

func TestNewMetricsCollector(t *testing.T) {
	m := NewMetricsCollector()
	if m == nil {
		t.Fatal("NewMetricsCollector should return non-nil")
	}
	if m.startTime.IsZero() {
		t.Error("startTime should be set")
	}
}

func TestMetricsCollector_RecordHit(t *testing.T) {
	m := NewMetricsCollector()
	m.RecordHit()
	m.RecordHit()
	m.RecordHit()

	snapshot := m.GetSnapshot(New())
	if snapshot.Hits != 3 {
		t.Errorf("expected 3 hits, got %d", snapshot.Hits)
	}
	if snapshot.TotalCommands != 3 {
		t.Errorf("expected 3 total commands, got %d", snapshot.TotalCommands)
	}
}

func TestMetricsCollector_RecordMiss(t *testing.T) {
	m := NewMetricsCollector()
	m.RecordMiss()
	m.RecordMiss()

	snapshot := m.GetSnapshot(New())
	if snapshot.Misses != 2 {
		t.Errorf("expected 2 misses, got %d", snapshot.Misses)
	}
}

func TestMetricsCollector_RecordSet(t *testing.T) {
	m := NewMetricsCollector()
	m.RecordSet()
	m.RecordSet()
	m.RecordSet()

	snapshot := m.GetSnapshot(New())
	if snapshot.Sets != 3 {
		t.Errorf("expected 3 sets, got %d", snapshot.Sets)
	}
}

func TestMetricsCollector_RecordDelete(t *testing.T) {
	m := NewMetricsCollector()
	m.RecordDelete()

	snapshot := m.GetSnapshot(New())
	if snapshot.Deletes != 1 {
		t.Errorf("expected 1 delete, got %d", snapshot.Deletes)
	}
}

func TestMetricsCollector_RecordExpired(t *testing.T) {
	m := NewMetricsCollector()
	m.RecordExpired(5)

	snapshot := m.GetSnapshot(New())
	if snapshot.ExpiredCount != 5 {
		t.Errorf("expected 5 expired, got %d", snapshot.ExpiredCount)
	}
}

func TestMetricsCollector_RecordEviction(t *testing.T) {
	m := NewMetricsCollector()
	m.RecordEviction()
	m.RecordEviction()

	snapshot := m.GetSnapshot(New())
	if snapshot.EvictedCount != 2 {
		t.Errorf("expected 2 evicted, got %d", snapshot.EvictedCount)
	}
}

func TestMetricsCollector_ConnectedClients(t *testing.T) {
	m := NewMetricsCollector()

	m.AddConnectedClient()
	m.AddConnectedClient()
	m.AddConnectedClient()

	snapshot := m.GetSnapshot(New())
	if snapshot.ConnectedClients != 3 {
		t.Errorf("expected 3 connected clients, got %d", snapshot.ConnectedClients)
	}

	m.RemoveConnectedClient()

	snapshot = m.GetSnapshot(New())
	if snapshot.ConnectedClients != 2 {
		t.Errorf("expected 2 connected clients after removal, got %d", snapshot.ConnectedClients)
	}
}

func TestMetricsCollector_SetConnectedClients(t *testing.T) {
	m := NewMetricsCollector()
	m.SetConnectedClients(10)

	snapshot := m.GetSnapshot(New())
	if snapshot.ConnectedClients != 10 {
		t.Errorf("expected 10 connected clients, got %d", snapshot.ConnectedClients)
	}
}

func TestMetricsCollector_RecordCommand(t *testing.T) {
	m := NewMetricsCollector()
	m.RecordCommand()
	m.RecordCommand()

	snapshot := m.GetSnapshot(New())
	if snapshot.TotalCommands != 2 {
		t.Errorf("expected 2 total commands, got %d", snapshot.TotalCommands)
	}
}

func TestMetricsCollector_HitRate(t *testing.T) {
	m := NewMetricsCollector()

	m.RecordHit()
	m.RecordHit()
	m.RecordHit()
	m.RecordMiss()

	snapshot := m.GetSnapshot(New())
	if snapshot.HitRate != 75.0 {
		t.Errorf("expected 75%% hit rate, got %.2f%%", snapshot.HitRate)
	}
}

func TestMetricsCollector_HitRateZero(t *testing.T) {
	m := NewMetricsCollector()

	snapshot := m.GetSnapshot(New())
	if snapshot.HitRate != 0.0 {
		t.Errorf("expected 0%% hit rate with no ops, got %.2f%%", snapshot.HitRate)
	}
}

func TestMetricsCollector_Uptime(t *testing.T) {
	m := NewMetricsCollector()

	snapshot := m.GetSnapshot(New())
	if snapshot.UptimeSeconds < 0 {
		t.Errorf("uptime should be non-negative, got %f", snapshot.UptimeSeconds)
	}
}

func TestMetricsCollector_KeysCount(t *testing.T) {
	m := NewMetricsCollector()
	c := New()
	c.Set("k1", "v1", 0)
	c.Set("k2", "v2", 0)

	snapshot := m.GetSnapshot(c)
	if snapshot.KeysCount != 2 {
		t.Errorf("expected 2 keys, got %d", snapshot.KeysCount)
	}
}

func TestMetricsCollector_PrometheusFormat(t *testing.T) {
	m := NewMetricsCollector()
	c := New()
	c.Set("test_key", "test_value", 0)

	m.RecordHit()
	m.RecordSet()

	output := m.PrometheusFormat(c)

	expectedMetrics := []string{
		"gocache_hits",
		"gocache_misses",
		"gocache_hit_rate",
		"gocache_sets",
		"gocache_deletes",
		"gocache_expired",
		"gocache_evicted",
		"gocache_connected_clients",
		"gocache_total_commands",
		"gocache_keys",
		"gocache_used_memory_bytes",
		"gocache_max_memory_bytes",
		"gocache_uptime_seconds",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(output, metric) {
			t.Errorf("Prometheus output missing metric: %s", metric)
		}
	}

	if !strings.Contains(output, "# HELP") {
		t.Error("Prometheus output should contain HELP comments")
	}
	if !strings.Contains(output, "# TYPE") {
		t.Error("Prometheus output should contain TYPE comments")
	}
}

func TestMetricsCollector_Concurrent(t *testing.T) {
	m := NewMetricsCollector()

	done := make(chan struct{})
	for i := 0; i < 100; i++ {
		go func() {
			m.RecordHit()
			m.RecordMiss()
			m.RecordSet()
			m.RecordDelete()
			m.RecordCommand()
			m.AddConnectedClient()
			done <- struct{}{}
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	for i := 0; i < 100; i++ {
		m.RemoveConnectedClient()
	}

	snapshot := m.GetSnapshot(New())
	if snapshot.Hits != 100 {
		t.Errorf("expected 100 hits, got %d", snapshot.Hits)
	}
	if snapshot.ConnectedClients != 0 {
		t.Errorf("expected 0 connected clients, got %d", snapshot.ConnectedClients)
	}
}

func TestMetricsSnapshot_JSON(t *testing.T) {
	m := NewMetricsCollector()
	c := New()
	c.Set("k1", "v1", 0)

	m.RecordHit()
	m.RecordMiss()

	snapshot := m.GetSnapshot(c)

	if snapshot.Hits != 1 {
		t.Errorf("expected 1 hit, got %d", snapshot.Hits)
	}
	if snapshot.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", snapshot.Misses)
	}
	if snapshot.KeysCount != 1 {
		t.Errorf("expected 1 key, got %d", snapshot.KeysCount)
	}
	if snapshot.UptimeSeconds < 0 {
		t.Errorf("uptime should be non-negative, got %f", snapshot.UptimeSeconds)
	}
}

func TestMetricsCollector_MemoryUsage(t *testing.T) {
	m := NewMetricsCollector()
	c := New()
	c.Set("key1", "value1", 0)

	snapshot := m.GetSnapshot(c)
	if snapshot.UsedMemory <= 0 {
		t.Errorf("used memory should be positive, got %d", snapshot.UsedMemory)
	}
}

func TestMetricsCollector_StartTime(t *testing.T) {
	m := NewMetricsCollector()
	before := time.Now()
	time.Sleep(1 * time.Millisecond)
	after := time.Now()

	if m.startTime.Before(before) || m.startTime.After(after) {
		t.Error("startTime should be between before and after")
	}
}
