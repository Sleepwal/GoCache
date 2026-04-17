package cache

import (
	"fmt"
	"sync"
	"time"

	"GoCache/logger"
)

// PubSubEvent 发布/订阅事件类型
type PubSubEvent int

const (
	EventSet PubSubEvent = iota
	EventDelete
	EventExpire
	EventClear
)

func (e PubSubEvent) String() string {
	switch e {
	case EventSet:
		return "SET"
	case EventDelete:
		return "DELETE"
	case EventExpire:
		return "EXPIRE"
	case EventClear:
		return "CLEAR"
	default:
		return "UNKNOWN"
	}
}

// CacheEvent 缓存事件
type CacheEvent struct {
	Type      PubSubEvent
	Key       string
	Value     any
	Timestamp time.Time
}

// Subscription 订阅
type Subscription struct {
	id      string
	channel chan CacheEvent
	closed  bool
}

// PubSub 发布/订阅系统
type PubSub struct {
	mu            sync.RWMutex
	subscribers   map[string][]*Subscription // key -> subscribers
	globalSubs    []*Subscription            // 全局订阅（接收所有事件）
	counter       int
}

// NewPubSub 创建发布/订阅系统
func NewPubSub() *PubSub {
	return &PubSub{
		subscribers: make(map[string][]*Subscription),
	}
}

// Subscribe 订阅特定键的事件
func (ps *PubSub) Subscribe(keys ...string) *Subscription {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.counter++
	sub := &Subscription{
		id:      fmt.Sprintf("sub-%d", ps.counter),
		channel: make(chan CacheEvent, 100),
	}

	if len(keys) == 0 {
		ps.globalSubs = append(ps.globalSubs, sub)
		logger.Debug("pubsub global subscription created", "sub_id", sub.id)
	} else {
		for _, key := range keys {
			ps.subscribers[key] = append(ps.subscribers[key], sub)
		}
		logger.Debug("pubsub subscription created", "sub_id", sub.id, "keys", keys)
	}

	return sub
}

// Unsubscribe 取消订阅
func (ps *PubSub) Unsubscribe(sub *Subscription) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if sub.closed {
		return
	}

	sub.closed = true
	close(sub.channel)
	logger.Debug("pubsub unsubscribed", "sub_id", sub.id)

	for i, s := range ps.globalSubs {
		if s == sub {
			ps.globalSubs = append(ps.globalSubs[:i], ps.globalSubs[i+1:]...)
			break
		}
	}

	// 从键订阅中移除
	for key, subs := range ps.subscribers {
		for i, s := range subs {
			if s == sub {
				ps.subscribers[key] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
	}
}

// Publish 发布事件
func (ps *PubSub) Publish(event CacheEvent) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	event.Timestamp = time.Now()

	for _, sub := range ps.globalSubs {
		if !sub.closed {
			select {
			case sub.channel <- event:
			default:
				logger.Warn("pubsub channel full, event dropped", "sub_id", sub.id, "event_type", event.Type)
			}
		}
	}

	for _, sub := range ps.subscribers[event.Key] {
		if !sub.closed {
			select {
			case sub.channel <- event:
			default:
				logger.Warn("pubsub channel full, event dropped", "sub_id", sub.id, "key", event.Key, "event_type", event.Type)
			}
		}
	}
}

// Channel 返回订阅通道
func (s *Subscription) Channel() <-chan CacheEvent {
	return s.channel
}

// PubSubCache 带发布/订阅功能的缓存包装器
type PubSubCache struct {
	cache  *MemoryCache
	pubsub *PubSub
}

// NewPubSubCache 创建带发布/订阅功能的缓存
func NewPubSubCache(cache *MemoryCache) *PubSubCache {
	if cache == nil {
		cache = New()
	}
	return &PubSubCache{
		cache:  cache,
		pubsub: NewPubSub(),
	}
}

// Set 添加或更新缓存项并发布事件
func (pc *PubSubCache) Set(key string, value any, ttl time.Duration) {
	pc.cache.Set(key, value, ttl)
	pc.pubsub.Publish(CacheEvent{
		Type:  EventSet,
		Key:   key,
		Value: value,
	})
}

// Get 获取缓存项
func (pc *PubSubCache) Get(key string) (any, bool) {
	return pc.cache.Get(key)
}

// Delete 删除缓存项并发布事件
func (pc *PubSubCache) Delete(key string) bool {
	deleted := pc.cache.Delete(key)
	if deleted {
		pc.pubsub.Publish(CacheEvent{
			Type: EventDelete,
			Key:  key,
		})
	}
	return deleted
}

// Clear 清空缓存并发布事件
func (pc *PubSubCache) Clear() {
	pc.cache.Clear()
	pc.pubsub.Publish(CacheEvent{
		Type: EventClear,
	})
}

// Subscribe 订阅事件
func (pc *PubSubCache) Subscribe(keys ...string) *Subscription {
	return pc.pubsub.Subscribe(keys...)
}

// Unsubscribe 取消订阅
func (pc *PubSubCache) Unsubscribe(sub *Subscription) {
	pc.pubsub.Unsubscribe(sub)
}

// GetPubSub 获取内部 PubSub 实例
func (pc *PubSubCache) GetPubSub() *PubSub {
	return pc.pubsub
}

// GetCache 获取内部缓存实例
func (pc *PubSubCache) GetCache() *MemoryCache {
	return pc.cache
}
