package cache

import (
	"container/heap"
	"container/list"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
)

// CacheSnapshot 缓存快照数据结构
type CacheSnapshot struct {
	Items map[string]SnapshotItem `json:"items"`
}

// SnapshotItem 快照中的缓存项
type SnapshotItem struct {
	Value      interface{} `json:"value"`
	Expiration int64       `json:"expiration"`
}

// SaveToFile 将 MemoryCache 状态保存到文件（JSON 格式）
func (c *MemoryCache) SaveToFile(path string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	snapshot := CacheSnapshot{
		Items: make(map[string]SnapshotItem, len(c.items)),
	}

	for key, item := range c.items {
		snapshot.Items[key] = SnapshotItem{
			Value:      item.Value,
			Expiration: item.Expiration,
		}
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// LoadFromFile 从文件恢复 MemoryCache 状态
func (c *MemoryCache) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read snapshot file: %w", err)
	}

	var snapshot CacheSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*Item, len(snapshot.Items))
	for key, item := range snapshot.Items {
		c.items[key] = &Item{
			Value:      item.Value,
			Expiration: item.Expiration,
		}
	}

	return nil
}

// SaveToFileGob 使用 gob 格式保存快照（更高效，但仅限 Go 使用）
func (c *MemoryCache) SaveToFileGob(path string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)

	snapshot := CacheSnapshot{
		Items: make(map[string]SnapshotItem, len(c.items)),
	}

	for key, item := range c.items {
		snapshot.Items[key] = SnapshotItem{
			Value:      item.Value,
			Expiration: item.Expiration,
		}
	}

	return encoder.Encode(snapshot)
}

// LoadFromFileGob 使用 gob 格式加载快照
func (c *MemoryCache) LoadFromFileGob(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var snapshot CacheSnapshot
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&snapshot); err != nil {
		return fmt.Errorf("failed to decode snapshot: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*Item, len(snapshot.Items))
	for key, item := range snapshot.Items {
		c.items[key] = &Item{
			Value:      item.Value,
			Expiration: item.Expiration,
		}
	}

	return nil
}

// SaveToFile 将 LRUCache 状态保存到文件
func (c *LRUCache) SaveToFile(path string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	type LRUSnapshotItem struct {
		Value      interface{} `json:"value"`
		Expiration int64       `json:"expiration"`
	}

	snapshot := struct {
		Capacity int                    `json:"capacity"`
		Items    map[string]LRUSnapshotItem `json:"items"`
		Order    []string               `json:"order"`
	}{
		Capacity: c.capacity,
		Items:    make(map[string]LRUSnapshotItem),
		Order:    make([]string, 0, len(c.items)),
	}

	// 保存链表顺序
	for elem := c.lruList.Front(); elem != nil; elem = elem.Next() {
		item := elem.Value.(*lruItem)
		snapshot.Items[item.key] = LRUSnapshotItem{
			Value:      item.value,
			Expiration: item.expiration,
		}
		snapshot.Order = append(snapshot.Order, item.key)
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal LRU snapshot: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// LoadFromFile 从文件恢复 LRUCache 状态
func (c *LRUCache) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read LRU snapshot file: %w", err)
	}

	type LRUSnapshotItem struct {
		Value      interface{} `json:"value"`
		Expiration int64       `json:"expiration"`
	}

	var snapshot struct {
		Capacity int                        `json:"capacity"`
		Items    map[string]LRUSnapshotItem `json:"items"`
		Order    []string                   `json:"order"`
	}

	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fmt.Errorf("failed to unmarshal LRU snapshot: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.capacity = snapshot.Capacity
	c.items = make(map[string]*list.Element, len(snapshot.Items))
	c.lruList = list.New()

	// 按照保存的顺序重建链表
	for _, key := range snapshot.Order {
		item := snapshot.Items[key]
		newItem := &lruItem{
			key:        key,
			value:      item.Value,
			expiration: item.Expiration,
		}
		elem := c.lruList.PushBack(newItem)
		c.items[key] = elem
	}

	return nil
}

// SaveToFile 将 LFUCache 状态保存到文件
func (c *LFUCache) SaveToFile(path string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	type LFUSnapshotItem struct {
		Value      interface{} `json:"value"`
		Expiration int64       `json:"expiration"`
		Frequency  float64     `json:"frequency"`
	}

	snapshot := struct {
		Capacity    int                       `json:"capacity"`
		Items       map[string]LFUSnapshotItem `json:"items"`
		DecayFactor float64                   `json:"decay_factor"`
	}{
		Capacity:    c.capacity,
		Items:       make(map[string]LFUSnapshotItem, len(c.items)),
		DecayFactor: c.decayFactor,
	}

	for key, item := range c.items {
		snapshot.Items[key] = LFUSnapshotItem{
			Value:      item.value,
			Expiration: item.expiration,
			Frequency:  item.frequency,
		}
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal LFU snapshot: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// LoadFromFile 从文件恢复 LFUCache 状态
func (c *LFUCache) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read LFU snapshot file: %w", err)
	}

	type LFUSnapshotItem struct {
		Value      interface{} `json:"value"`
		Expiration int64       `json:"expiration"`
		Frequency  float64     `json:"frequency"`
	}

	var snapshot struct {
		Capacity    int                        `json:"capacity"`
		Items       map[string]LFUSnapshotItem `json:"items"`
		DecayFactor float64                    `json:"decay_factor"`
	}

	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fmt.Errorf("failed to unmarshal LFU snapshot: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.capacity = snapshot.Capacity
	c.decayFactor = snapshot.DecayFactor
	c.items = make(map[string]*lfuItem, len(snapshot.Items))
	c.freqHeap = &frequencyHeap{}

	for key, item := range snapshot.Items {
		newItem := &lfuItem{
			key:        key,
			value:      item.Value,
			expiration: item.Expiration,
			frequency:  item.Frequency,
		}
		c.items[key] = newItem
		heap.Push(c.freqHeap, newItem)
	}

	return nil
}
