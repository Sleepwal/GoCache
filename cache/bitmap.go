package cache

import (
	"errors"
	"time"
)

var (
	ErrInvalidOffset  = errors.New("ERR bit offset is not an integer or out of range")
	ErrNotEnoughArgs  = errors.New("ERR wrong number of arguments")
	ErrBitopNotSingle = errors.New("ERR BITOP NOT must be called with a single source key")
	ErrUnknownBitOp   = errors.New("ERR syntax error")
)

type bitmapData struct {
	bits []byte
	len  int
}

func newBitmapData() *bitmapData {
	return &bitmapData{
		bits: make([]byte, 0),
		len:  0,
	}
}

func (bd *bitmapData) ensureCapacity(offset int) {
	byteIndex := offset / 8
	for byteIndex >= len(bd.bits) {
		bd.bits = append(bd.bits, 0)
	}
	if offset+1 > bd.len {
		bd.len = offset + 1
	}
}

type BitmapCache struct {
	cache *MemoryCache
}

func NewBitmapCache(cache *MemoryCache) *BitmapCache {
	if cache == nil {
		cache = New()
	}
	return &BitmapCache{cache: cache}
}

func (bc *BitmapCache) getOrCreateBitmap(key string) (*bitmapData, bool) {
	val, found := bc.cache.Get(key)
	if !found {
		return nil, false
	}
	bd, ok := val.(*bitmapData)
	if !ok {
		return nil, false
	}
	return bd, true
}

func (bc *BitmapCache) SetBit(key string, offset int, value int) (int, error) {
	bc.cache.mu.Lock()
	defer bc.cache.mu.Unlock()

	if offset < 0 {
		return 0, ErrInvalidOffset
	}

	var bd *bitmapData
	item, exists := bc.cache.items[key]
	if exists && !item.IsExpired() {
		var ok bool
		bd, ok = item.Value.(*bitmapData)
		if !ok {
			bd = newBitmapData()
		}
	} else {
		bd = newBitmapData()
	}

	bd.ensureCapacity(offset)

	byteIndex := offset / 8
	bitIndex := uint(offset % 8)

	oldBit := int((bd.bits[byteIndex] >> bitIndex) & 1)

	if value == 1 {
		bd.bits[byteIndex] |= 1 << bitIndex
	} else {
		bd.bits[byteIndex] &^= 1 << bitIndex
	}

	bc.cache.items[key] = &Item{
		Value:      bd,
		Expiration: 0,
		LastAccess: time.Now().UnixNano(),
	}

	return oldBit, nil
}

func (bc *BitmapCache) GetBit(key string, offset int) (int, error) {
	if offset < 0 {
		return 0, ErrInvalidOffset
	}

	bc.cache.mu.RLock()
	defer bc.cache.mu.RUnlock()

	bd, found := bc.getOrCreateBitmapRLocked(key)
	if !found {
		return 0, nil
	}

	byteIndex := offset / 8
	if byteIndex >= len(bd.bits) {
		return 0, nil
	}

	bitIndex := uint(offset % 8)
	return int((bd.bits[byteIndex] >> bitIndex) & 1), nil
}

func (bc *BitmapCache) getOrCreateBitmapRLocked(key string) (*bitmapData, bool) {
	item, exists := bc.cache.items[key]
	if !exists || item.IsExpired() {
		return nil, false
	}
	bd, ok := item.Value.(*bitmapData)
	if !ok {
		return nil, false
	}
	return bd, true
}

func (bc *BitmapCache) BitCount(key string, start, end int) (int, error) {
	bc.cache.mu.RLock()
	defer bc.cache.mu.RUnlock()

	bd, found := bc.getOrCreateBitmapRLocked(key)
	if !found {
		return 0, nil
	}

	return bc.bitCountRange(bd, start, end), nil
}

func (bc *BitmapCache) bitCountRange(bd *bitmapData, start, end int) int {
	if start < 0 {
		start = len(bd.bits) + start
	}
	if end < 0 {
		end = len(bd.bits) + end
	}
	if start < 0 {
		start = 0
	}
	if end >= len(bd.bits) {
		end = len(bd.bits) - 1
	}
	if start > end || len(bd.bits) == 0 {
		return 0
	}

	count := 0
	for i := start; i <= end; i++ {
		count += popcount(bd.bits[i])
	}
	return count
}

func (bc *BitmapCache) BitCountAll(key string) (int, error) {
	bc.cache.mu.RLock()
	defer bc.cache.mu.RUnlock()

	bd, found := bc.getOrCreateBitmapRLocked(key)
	if !found {
		return 0, nil
	}

	count := 0
	for _, b := range bd.bits {
		count += popcount(b)
	}
	return count, nil
}

func (bc *BitmapCache) BitOp(operation, destKey string, srcKeys ...string) (int, error) {
	bc.cache.mu.Lock()
	defer bc.cache.mu.Unlock()

	if len(srcKeys) == 0 {
		return 0, ErrNotEnoughArgs
	}

	srcBitmaps := make([]*bitmapData, 0, len(srcKeys))
	maxLen := 0

	for _, key := range srcKeys {
		item, exists := bc.cache.items[key]
		if !exists || item.IsExpired() {
			srcBitmaps = append(srcBitmaps, newBitmapData())
			continue
		}
		bd, ok := item.Value.(*bitmapData)
		if !ok {
			srcBitmaps = append(srcBitmaps, newBitmapData())
			continue
		}
		srcBitmaps = append(srcBitmaps, bd)
		if len(bd.bits) > maxLen {
			maxLen = len(bd.bits)
		}
	}

	result := make([]byte, maxLen)

	switch stringsToUpper(operation) {
	case "AND":
		for i := 0; i < maxLen; i++ {
			result[i] = 0xFF
			for _, bd := range srcBitmaps {
				if i < len(bd.bits) {
					result[i] &= bd.bits[i]
				} else {
					result[i] &= 0
				}
			}
		}
	case "OR":
		for i := 0; i < maxLen; i++ {
			result[i] = 0
			for _, bd := range srcBitmaps {
				if i < len(bd.bits) {
					result[i] |= bd.bits[i]
				}
			}
		}
	case "XOR":
		for i := 0; i < maxLen; i++ {
			result[i] = 0
			for _, bd := range srcBitmaps {
				if i < len(bd.bits) {
					result[i] ^= bd.bits[i]
				}
			}
		}
	case "NOT":
		if len(srcKeys) != 1 {
			return 0, ErrBitopNotSingle
		}
		bd := srcBitmaps[0]
		for i := 0; i < maxLen; i++ {
			if i < len(bd.bits) {
				result[i] = ^bd.bits[i]
			} else {
				result[i] = 0xFF
			}
		}
	default:
		return 0, ErrUnknownBitOp
	}

	dest := &bitmapData{bits: result, len: maxLen * 8}
	bc.cache.items[destKey] = &Item{
		Value:      dest,
		Expiration: 0,
		LastAccess: time.Now().UnixNano(),
	}

	return maxLen * 8, nil
}

func (bc *BitmapCache) BitPos(key string, bit int, start, end int, endGiven bool) (int, error) {
	bc.cache.mu.RLock()
	defer bc.cache.mu.RUnlock()

	bd, found := bc.getOrCreateBitmapRLocked(key)
	if !found {
		if bit == 1 {
			return -1, nil
		}
		return 0, nil
	}

	if start < 0 {
		start = len(bd.bits) + start
	}
	if start < 0 {
		start = 0
	}

	if !endGiven {
		end = len(bd.bits) - 1
	} else {
		if end < 0 {
			end = len(bd.bits) + end
		}
		if end >= len(bd.bits) {
			end = len(bd.bits) - 1
		}
	}

	if start > end || len(bd.bits) == 0 {
		return -1, nil
	}

	for i := start; i <= end; i++ {
		if bit == 1 {
			if bd.bits[i] != 0 {
				for b := 0; b < 8; b++ {
					if (bd.bits[i]>>uint(b))&1 == 1 {
						return i*8 + b, nil
					}
				}
			}
		} else {
			if bd.bits[i] != 0xFF {
				for b := 0; b < 8; b++ {
					if (bd.bits[i]>>uint(b))&1 == 0 {
						return i*8 + b, nil
					}
				}
			}
		}
	}

	if bit == 0 && start == 0 && end == len(bd.bits)-1 {
		lastBit := bd.len
		if lastBit > 0 {
			return lastBit, nil
		}
	}

	return -1, nil
}

func (bc *BitmapCache) StringLen(key string) (int, error) {
	bc.cache.mu.RLock()
	defer bc.cache.mu.RUnlock()

	bd, found := bc.getOrCreateBitmapRLocked(key)
	if !found {
		return 0, nil
	}
	return len(bd.bits), nil
}

func popcount(b byte) int {
	count := 0
	for b != 0 {
		count += int(b & 1)
		b >>= 1
	}
	return count
}

func stringsToUpper(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			result[i] = c - 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}
