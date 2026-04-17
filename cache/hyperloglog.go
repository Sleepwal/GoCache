package cache

import (
	"hash/fnv"
	"math"
	"time"
)

const (
	hllP        = 14
	hllM        = 1 << hllP
	hllAlphaInf = 0.7213 / (1.0 + 1.079/float64(hllM))
)

type hllData struct {
	registers [hllM]byte
	cache     *MemoryCache
}

func newHLLData() *hllData {
	return &hllData{}
}

func (h *hllData) hash(val string) uint64 {
	hf := fnv.New64a()
	hf.Write([]byte(val))
	return hf.Sum64()
}

func (h *hllData) add(val string) bool {
	hash := h.hash(val)

	idx := hash & (hllM - 1)
	rho := uint8(1)
	w := hash >> hllP
	for w != 0 && rho <= 64-hllP {
		rho++
		w >>= 1
	}

	if h.registers[idx] < rho {
		h.registers[idx] = rho
		return true
	}
	return false
}

func (h *hllData) count() int64 {
	sum := 0.0
	zeroCount := 0

	for i := 0; i < hllM; i++ {
		sum += math.Pow(2.0, -float64(h.registers[i]))
		if h.registers[i] == 0 {
			zeroCount++
		}
	}

	estimate := hllAlphaInf * float64(hllM) * float64(hllM) / sum

	if estimate <= 2.5*float64(hllM) && zeroCount > 0 {
		estimate = float64(hllM) * math.Log(float64(hllM)/float64(zeroCount))
	}

	if estimate > math.Pow(2.0, 32) {
		estimate = math.Pow(2.0, 32)
	}

	return int64(estimate + 0.5)
}

func (h *hllData) merge(other *hllData) {
	for i := 0; i < hllM; i++ {
		if other.registers[i] > h.registers[i] {
			h.registers[i] = other.registers[i]
		}
	}
}

type HyperLogLogCache struct {
	cache *MemoryCache
}

func NewHyperLogLogCache(cache *MemoryCache) *HyperLogLogCache {
	if cache == nil {
		cache = New()
	}
	return &HyperLogLogCache{cache: cache}
}

func (hlc *HyperLogLogCache) getOrCreateHLL(key string) (*hllData, bool) {
	val, found := hlc.cache.Get(key)
	if !found {
		return nil, false
	}
	hd, ok := val.(*hllData)
	if !ok {
		return nil, false
	}
	return hd, true
}

func (hlc *HyperLogLogCache) PFAdd(key string, elements ...string) (int, error) {
	hlc.cache.mu.Lock()
	defer hlc.cache.mu.Unlock()

	var hd *hllData
	item, exists := hlc.cache.items[key]
	if exists && !item.IsExpired() {
		var ok bool
		hd, ok = item.Value.(*hllData)
		if !ok {
			hd = newHLLData()
		}
	} else {
		hd = newHLLData()
	}

	updated := 0
	for _, elem := range elements {
		if hd.add(elem) {
			updated = 1
		}
	}

	hlc.cache.items[key] = &Item{
		Value:      hd,
		Expiration: 0,
		LastAccess: time.Now().UnixNano(),
	}

	return updated, nil
}

func (hlc *HyperLogLogCache) PFCount(keys ...string) (int64, error) {
	hlc.cache.mu.RLock()
	defer hlc.cache.mu.RUnlock()

	if len(keys) == 0 {
		return 0, nil
	}

	if len(keys) == 1 {
		hd, found := hlc.getOrCreateHLLRLocked(keys[0])
		if !found {
			return 0, nil
		}
		return hd.count(), nil
	}

	merged := newHLLData()
	for _, key := range keys {
		hd, found := hlc.getOrCreateHLLRLocked(key)
		if !found {
			continue
		}
		merged.merge(hd)
	}

	return merged.count(), nil
}

func (hlc *HyperLogLogCache) getOrCreateHLLRLocked(key string) (*hllData, bool) {
	item, exists := hlc.cache.items[key]
	if !exists || item.IsExpired() {
		return nil, false
	}
	hd, ok := item.Value.(*hllData)
	if !ok {
		return nil, false
	}
	return hd, true
}

func (hlc *HyperLogLogCache) PFMerge(destKey string, srcKeys ...string) (string, error) {
	hlc.cache.mu.Lock()
	defer hlc.cache.mu.Unlock()

	if len(srcKeys) == 0 {
		return "OK", nil
	}

	merged := newHLLData()
	for _, key := range srcKeys {
		item, exists := hlc.cache.items[key]
		if !exists || item.IsExpired() {
			continue
		}
		hd, ok := item.Value.(*hllData)
		if !ok {
			continue
		}
		merged.merge(hd)
	}

	hlc.cache.items[destKey] = &Item{
		Value:      merged,
		Expiration: 0,
		LastAccess: time.Now().UnixNano(),
	}

	return "OK", nil
}
