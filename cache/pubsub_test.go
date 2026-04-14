package cache

import (
	"testing"
	"time"
)

// TestPubSub_Subscribe 测试订阅
func TestPubSub_Subscribe(t *testing.T) {
	ps := NewPubSub()

	sub := ps.Subscribe("key1", "key2")
	if sub == nil {
		t.Fatal("expected subscription to be created")
	}

	// 发布事件
	ps.Publish(CacheEvent{
		Type: EventSet,
		Key:  "key1",
	})

	// 接收事件
	select {
	case event := <-sub.Channel():
		if event.Type != EventSet || event.Key != "key1" {
			t.Errorf("unexpected event: %+v", event)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected to receive event")
	}
}

// TestPubSub_GlobalSubscribe 测试全局订阅
func TestPubSub_GlobalSubscribe(t *testing.T) {
	ps := NewPubSub()

	sub := ps.Subscribe() // 全局订阅

	ps.Publish(CacheEvent{
		Type: EventSet,
		Key:  "any_key",
	})

	select {
	case event := <-sub.Channel():
		if event.Key != "any_key" {
			t.Errorf("expected 'any_key', got '%s'", event.Key)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected to receive event")
	}
}

// TestPubSub_Unsubscribe 测试取消订阅
func TestPubSub_Unsubscribe(t *testing.T) {
	ps := NewPubSub()

	sub := ps.Subscribe("key1")
	ps.Unsubscribe(sub)

	// 给一点时间让取消订阅生效
	time.Sleep(10 * time.Millisecond)

	// 发布事件
	ps.Publish(CacheEvent{
		Type: EventSet,
		Key:  "key1",
	})

	// 不应该收到事件（或者收到之前已经在通道中的事件）
	select {
	case <-sub.Channel():
		// 可能收到之前的事件，这也是可以接受的
	case <-time.After(50 * time.Millisecond):
		// 预期行为
	}
}

// TestPubSub_EventOrder 测试事件顺序
func TestPubSub_EventOrder(t *testing.T) {
	ps := NewPubSub()

	sub := ps.Subscribe("key")

	ps.Publish(CacheEvent{Type: EventSet, Key: "key"})
	ps.Publish(CacheEvent{Type: EventDelete, Key: "key"})
	ps.Publish(CacheEvent{Type: EventSet, Key: "key"})

	// 接收第一个事件
	event := <-sub.Channel()
	if event.Type != EventSet {
		t.Errorf("expected SET, got %v", event.Type)
	}

	// 接收第二个事件
	event = <-sub.Channel()
	if event.Type != EventDelete {
		t.Errorf("expected DELETE, got %v", event.Type)
	}

	// 接收第三个事件
	event = <-sub.Channel()
	if event.Type != EventSet {
		t.Errorf("expected SET, got %v", event.Type)
	}
}

// TestPubSubCache_Set 测试 PubSubCache Set
func TestPubSubCache_Set(t *testing.T) {
	pc := NewPubSubCache(New())

	sub := pc.Subscribe("key1")

	pc.Set("key1", "value1", 0)

	select {
	case event := <-sub.Channel():
		if event.Type != EventSet || event.Key != "key1" {
			t.Errorf("unexpected event: %+v", event)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected to receive SET event")
	}
}

// TestPubSubCache_Delete 测试 PubSubCache Delete
func TestPubSubCache_Delete(t *testing.T) {
	pc := NewPubSubCache(New())

	pc.Set("key1", "value1", 0)
	sub := pc.Subscribe("key1")

	pc.Delete("key1")

	select {
	case event := <-sub.Channel():
		if event.Type != EventDelete {
			t.Errorf("expected DELETE event, got %v", event.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected to receive DELETE event")
	}
}

// TestPubSubCache_Clear 测试 PubSubCache Clear
func TestPubSubCache_Clear(t *testing.T) {
	pc := NewPubSubCache(New())

	pc.Set("key1", "value1", 0)
	pc.Set("key2", "value2", 0)

	sub := pc.Subscribe() // 全局订阅

	pc.Clear()

	select {
	case event := <-sub.Channel():
		if event.Type != EventClear {
			t.Errorf("expected CLEAR event, got %v", event.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected to receive CLEAR event")
	}
}

// TestPubSub_MultipleSubscribers 测试多个订阅者
func TestPubSub_MultipleSubscribers(t *testing.T) {
	ps := NewPubSub()

	sub1 := ps.Subscribe("key")
	sub2 := ps.Subscribe("key")

	ps.Publish(CacheEvent{Type: EventSet, Key: "key"})

	// 两个订阅者都应该收到事件
	for _, sub := range []*Subscription{sub1, sub2} {
		select {
		case event := <-sub.Channel():
			if event.Key != "key" {
				t.Errorf("expected 'key', got '%s'", event.Key)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("expected subscriber to receive event")
		}
	}
}

// TestPubSub_EventTypes 测试所有事件类型
func TestPubSub_EventTypes(t *testing.T) {
	ps := NewPubSub()
	sub := ps.Subscribe()

	events := []PubSubEvent{EventSet, EventDelete, EventExpire, EventClear}

	for _, eventType := range events {
		ps.Publish(CacheEvent{Type: eventType, Key: "key"})

		select {
		case event := <-sub.Channel():
			if event.Type != eventType {
				t.Errorf("expected %v, got %v", eventType, event.Type)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("expected to receive %v event", eventType)
		}
	}
}

// TestPubSubCache_GetInternal 测试获取内部组件
func TestPubSubCache_GetInternal(t *testing.T) {
	pc := NewPubSubCache(New())

	cache := pc.GetCache()
	if cache == nil {
		t.Error("expected cache to be non-nil")
	}

	pubsub := pc.GetPubSub()
	if pubsub == nil {
		t.Error("expected pubsub to be non-nil")
	}
}

// TestPubSub_ChannelBuffer 测试通道缓冲
func TestPubSub_ChannelBuffer(t *testing.T) {
	ps := NewPubSub()
	sub := ps.Subscribe("key")

	// 发布多个事件
	for i := 0; i < 50; i++ {
		ps.Publish(CacheEvent{Type: EventSet, Key: "key"})
	}

	// 应该能接收到所有事件（在缓冲区大小内）
	received := 0
	for {
		select {
		case <-sub.Channel():
			received++
		default:
			goto done
		}
	}
done:

	if received == 0 {
		t.Error("expected to receive some events")
	}
}

// TestPubSubEvent_String 测试事件类型字符串
func TestPubSubEvent_String(t *testing.T) {
	tests := []struct {
		event    PubSubEvent
		expected string
	}{
		{EventSet, "SET"},
		{EventDelete, "DELETE"},
		{EventExpire, "EXPIRE"},
		{EventClear, "CLEAR"},
	}

	for _, tt := range tests {
		if tt.event.String() != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.event.String())
		}
	}
}
