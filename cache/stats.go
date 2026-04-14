package cache

import (
	"math"
	"sync/atomic"
)

// Stats 缓存统计指标
type Stats struct {
	// 基础指标
	Hits    atomic.Int64 // 命中次数
	Misses  atomic.Int64 // 未命中次数
	Sets    atomic.Int64 // 设置次数
	Deletes atomic.Int64 // 删除次数

	// 过期相关指标
	ExpiredCount atomic.Int64 // 过期删除次数
	TTLHits      atomic.Int64 // TTL 有效期内命中次数
	TTLMisses    atomic.Int64 // TTL 过期后未命中次数
}

// HitRate 计算命中率 (0.0 - 1.0)
func (s *Stats) HitRate() float64 {
	hits := s.Hits.Load()
	misses := s.Misses.Load()
	total := hits + misses
	if total == 0 {
		return 0.0
	}
	return float64(hits) / float64(total)
}

// HitRatePercent 计算命中率百分比 (0.0 - 100.0)
func (s *Stats) HitRatePercent() float64 {
	return math.Round(s.HitRate()*10000) / 100
}

// TotalOperations 返回总操作次数
func (s *Stats) TotalOperations() int64 {
	return s.Hits.Load() + s.Misses.Load() + s.Sets.Load() + s.Deletes.Load()
}

// Snapshot 获取当前统计快照（非原子性，用于报告）
type StatsSnapshot struct {
	Hits         int64
	Misses       int64
	Sets         int64
	Deletes      int64
	ExpiredCount int64
	TTLHits      int64
	TTLMisses    int64
	HitRate      float64
}

// GetSnapshot 获取统计快照
func (s *Stats) GetSnapshot() StatsSnapshot {
	return StatsSnapshot{
		Hits:         s.Hits.Load(),
		Misses:       s.Misses.Load(),
		Sets:         s.Sets.Load(),
		Deletes:      s.Deletes.Load(),
		ExpiredCount: s.ExpiredCount.Load(),
		TTLHits:      s.TTLHits.Load(),
		TTLMisses:    s.TTLMisses.Load(),
		HitRate:      s.HitRate(),
	}
}

// Reset 重置所有统计指标
func (s *Stats) Reset() {
	s.Hits.Store(0)
	s.Misses.Store(0)
	s.Sets.Store(0)
	s.Deletes.Store(0)
	s.ExpiredCount.Store(0)
	s.TTLHits.Store(0)
	s.TTLMisses.Store(0)
}
