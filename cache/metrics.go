package cache

import (
	"fmt"
	"sync/atomic"
	"time"
)

type MetricsCollector struct {
	hits             atomic.Int64
	misses           atomic.Int64
	sets             atomic.Int64
	deletes          atomic.Int64
	expiredCount     atomic.Int64
	evictedCount     atomic.Int64
	connectedClients atomic.Int64
	totalCommands    atomic.Int64
	startTime        time.Time
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		startTime: time.Now(),
	}
}

func (m *MetricsCollector) RecordHit() {
	m.hits.Add(1)
	m.totalCommands.Add(1)
}

func (m *MetricsCollector) RecordMiss() {
	m.misses.Add(1)
	m.totalCommands.Add(1)
}

func (m *MetricsCollector) RecordSet() {
	m.sets.Add(1)
	m.totalCommands.Add(1)
}

func (m *MetricsCollector) RecordDelete() {
	m.deletes.Add(1)
	m.totalCommands.Add(1)
}

func (m *MetricsCollector) RecordCommand() {
	m.totalCommands.Add(1)
}

func (m *MetricsCollector) RecordExpired(count int64) {
	m.expiredCount.Add(count)
}

func (m *MetricsCollector) RecordEviction() {
	m.evictedCount.Add(1)
}

func (m *MetricsCollector) SetConnectedClients(n int64) {
	m.connectedClients.Store(n)
}

func (m *MetricsCollector) AddConnectedClient() {
	m.connectedClients.Add(1)
}

func (m *MetricsCollector) RemoveConnectedClient() {
	m.connectedClients.Add(-1)
}

type MetricsSnapshot struct {
	Hits             int64   `json:"hits"`
	Misses           int64   `json:"misses"`
	Sets             int64   `json:"sets"`
	Deletes          int64   `json:"deletes"`
	ExpiredCount     int64   `json:"expired_count"`
	EvictedCount     int64   `json:"evicted_count"`
	ConnectedClients int64   `json:"connected_clients"`
	TotalCommands    int64   `json:"total_commands"`
	HitRate          float64 `json:"hit_rate"`
	UptimeSeconds    float64 `json:"uptime_seconds"`
	KeysCount        int     `json:"keys_count"`
	UsedMemory       int64   `json:"used_memory"`
	MaxMemory        int64   `json:"max_memory"`
}

func (m *MetricsCollector) GetSnapshot(cache *MemoryCache) MetricsSnapshot {
	hits := m.hits.Load()
	misses := m.misses.Load()
	total := hits + misses

	hitRate := 0.0
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	cache.mu.RLock()
	keysCount := len(cache.items)
	usedMemory := int64(cache.currentBytes)
	maxMemory := int64(cache.MaxMemoryBytes)
	cache.mu.RUnlock()

	return MetricsSnapshot{
		Hits:             hits,
		Misses:           misses,
		Sets:             m.sets.Load(),
		Deletes:          m.deletes.Load(),
		ExpiredCount:     m.expiredCount.Load(),
		EvictedCount:     m.evictedCount.Load(),
		ConnectedClients: m.connectedClients.Load(),
		TotalCommands:    m.totalCommands.Load(),
		HitRate:          hitRate,
		UptimeSeconds:    time.Since(m.startTime).Seconds(),
		KeysCount:        keysCount,
		UsedMemory:       usedMemory,
		MaxMemory:        maxMemory,
	}
}

func (m *MetricsCollector) PrometheusFormat(cache *MemoryCache) string {
	snapshot := m.GetSnapshot(cache)

	var result string
	result += fmt.Sprintf("# HELP gocache_hits Total number of cache hits\n")
	result += fmt.Sprintf("# TYPE gocache_hits counter\n")
	result += fmt.Sprintf("gocache_hits %d\n\n", snapshot.Hits)

	result += fmt.Sprintf("# HELP gocache_misses Total number of cache misses\n")
	result += fmt.Sprintf("# TYPE gocache_misses counter\n")
	result += fmt.Sprintf("gocache_misses %d\n\n", snapshot.Misses)

	result += fmt.Sprintf("# HELP gocache_hit_rate Cache hit rate percentage\n")
	result += fmt.Sprintf("# TYPE gocache_hit_rate gauge\n")
	result += fmt.Sprintf("gocache_hit_rate %.2f\n\n", snapshot.HitRate)

	result += fmt.Sprintf("# HELP gocache_sets Total number of SET operations\n")
	result += fmt.Sprintf("# TYPE gocache_sets counter\n")
	result += fmt.Sprintf("gocache_sets %d\n\n", snapshot.Sets)

	result += fmt.Sprintf("# HELP gocache_deletes Total number of DELETE operations\n")
	result += fmt.Sprintf("# TYPE gocache_deletes counter\n")
	result += fmt.Sprintf("gocache_deletes %d\n\n", snapshot.Deletes)

	result += fmt.Sprintf("# HELP gocache_expired Total number of expired keys\n")
	result += fmt.Sprintf("# TYPE gocache_expired counter\n")
	result += fmt.Sprintf("gocache_expired %d\n\n", snapshot.ExpiredCount)

	result += fmt.Sprintf("# HELP gocache_evicted Total number of evicted keys\n")
	result += fmt.Sprintf("# TYPE gocache_evicted counter\n")
	result += fmt.Sprintf("gocache_evicted %d\n\n", snapshot.EvictedCount)

	result += fmt.Sprintf("# HELP gocache_connected_clients Number of connected clients\n")
	result += fmt.Sprintf("# TYPE gocache_connected_clients gauge\n")
	result += fmt.Sprintf("gocache_connected_clients %d\n\n", snapshot.ConnectedClients)

	result += fmt.Sprintf("# HELP gocache_total_commands Total number of commands processed\n")
	result += fmt.Sprintf("# TYPE gocache_total_commands counter\n")
	result += fmt.Sprintf("gocache_total_commands %d\n\n", snapshot.TotalCommands)

	result += fmt.Sprintf("# HELP gocache_keys Number of keys in cache\n")
	result += fmt.Sprintf("# TYPE gocache_keys gauge\n")
	result += fmt.Sprintf("gocache_keys %d\n\n", snapshot.KeysCount)

	result += fmt.Sprintf("# HELP gocache_used_memory_bytes Used memory in bytes\n")
	result += fmt.Sprintf("# TYPE gocache_used_memory_bytes gauge\n")
	result += fmt.Sprintf("gocache_used_memory_bytes %d\n\n", snapshot.UsedMemory)

	result += fmt.Sprintf("# HELP gocache_max_memory_bytes Maximum memory limit in bytes\n")
	result += fmt.Sprintf("# TYPE gocache_max_memory_bytes gauge\n")
	result += fmt.Sprintf("gocache_max_memory_bytes %d\n\n", snapshot.MaxMemory)

	result += fmt.Sprintf("# HELP gocache_uptime_seconds Server uptime in seconds\n")
	result += fmt.Sprintf("# TYPE gocache_uptime_seconds gauge\n")
	result += fmt.Sprintf("gocache_uptime_seconds %.0f\n", snapshot.UptimeSeconds)

	return result
}
