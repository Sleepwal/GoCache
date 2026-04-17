package cache

import (
	"time"

	"GoCache/logger"
)

func (c *MemoryCache) StartEviction(interval time.Duration) func() {
	stopCh := make(chan struct{})

	logger.Info("eviction scheduler started", "interval", interval.String())

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.deleteExpired()
			case <-stopCh:
				logger.Info("eviction scheduler stopped")
				return
			}
		}
	}()

	return func() {
		close(stopCh)
	}
}

func (c *MemoryCache) deleteExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().UnixNano()
	expired := 0

	for key, item := range c.items {
		if item.Expiration > 0 && now > item.Expiration {
			value := item.Value
			delete(c.items, key)
			expired++

			if c.onEvict != nil {
				c.onEvict(key, value, TTLExpired)
			}
		}
	}

	if expired > 0 {
		c.Stats.ExpiredCount.Add(int64(expired))
		logger.Debug("expired keys cleaned", "count", expired, "remaining", len(c.items))
	}
}

func (c *MemoryCache) DeleteExpired() {
	c.deleteExpired()
}
