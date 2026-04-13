package cache

import (
	"time"
)

// StartEviction 启动定期清理过期键的协程
// interval: 清理间隔时间
// 返回一个停止函数,调用该函数可以停止清理协程
func (c *MemoryCache) StartEviction(interval time.Duration) func() {
	stopCh := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.deleteExpired()
			case <-stopCh:
				return
			}
		}
	}()

	return func() {
		close(stopCh)
	}
}

// deleteExpired 删除所有过期的键
func (c *MemoryCache) deleteExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().UnixNano()

	for key, item := range c.items {
		if item.Expiration > 0 && now > item.Expiration {
			delete(c.items, key)
		}
	}
}

// DeleteExpired 公开方法:手动触发清理过期键
func (c *MemoryCache) DeleteExpired() {
	c.deleteExpired()
}
