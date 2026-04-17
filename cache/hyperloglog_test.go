package cache

import (
	"testing"
)

func TestHLL_PFAdd(t *testing.T) {
	c := New()
	hlc := NewHyperLogLogCache(c)

	result, err := hlc.PFAdd("hll", "a", "b", "c")
	if err != nil {
		t.Fatalf("PFAdd failed: %v", err)
	}
	if result != 1 {
		t.Errorf("expected 1 (at least one new element), got %d", result)
	}
}

func TestHLL_PFAddDuplicate(t *testing.T) {
	c := New()
	hlc := NewHyperLogLogCache(c)

	hlc.PFAdd("hll", "a", "b", "c")
	result, _ := hlc.PFAdd("hll", "a", "b", "c")

	if result != 0 {
		t.Errorf("expected 0 (no new elements), got %d", result)
	}
}

func TestHLL_PFCount(t *testing.T) {
	c := New()
	hlc := NewHyperLogLogCache(c)

	for i := 0; i < 1000; i++ {
		hlc.PFAdd("hll", string(rune(i)))
	}

	count, err := hlc.PFCount("hll")
	if err != nil {
		t.Fatalf("PFCount failed: %v", err)
	}

	errorRate := float64(count-1000) / 1000
	if errorRate < 0 {
		errorRate = -errorRate
	}
	if errorRate > 0.1 {
		t.Errorf("HLL count too far from expected: got %d, expected ~1000, error rate %.2f%%", count, errorRate*100)
	}
}

func TestHLL_PFCountNonExistent(t *testing.T) {
	c := New()
	hlc := NewHyperLogLogCache(c)

	count, err := hlc.PFCount("nonexistent")
	if err != nil {
		t.Fatalf("PFCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 for non-existent key, got %d", count)
	}
}

func TestHLL_PFCountMultiple(t *testing.T) {
	c := New()
	hlc := NewHyperLogLogCache(c)

	hlc.PFAdd("hll1", "a", "b", "c")
	hlc.PFAdd("hll2", "d", "e", "f")

	count, err := hlc.PFCount("hll1", "hll2")
	if err != nil {
		t.Fatalf("PFCount failed: %v", err)
	}
	if count < 5 {
		t.Errorf("expected at least 5 for union, got %d", count)
	}
}

func TestHLL_PFMerge(t *testing.T) {
	c := New()
	hlc := NewHyperLogLogCache(c)

	hlc.PFAdd("hll1", "a", "b", "c")
	hlc.PFAdd("hll2", "d", "e", "f")

	_, err := hlc.PFMerge("merged", "hll1", "hll2")
	if err != nil {
		t.Fatalf("PFMerge failed: %v", err)
	}

	count, _ := hlc.PFCount("merged")
	if count < 5 {
		t.Errorf("expected at least 5 after merge, got %d", count)
	}
}

func TestHLL_PFMergeOverlapping(t *testing.T) {
	c := New()
	hlc := NewHyperLogLogCache(c)

	hlc.PFAdd("hll1", "a", "b", "c")
	hlc.PFAdd("hll2", "b", "c", "d")

	hlc.PFMerge("merged", "hll1", "hll2")

	count, _ := hlc.PFCount("merged")
	if count < 3 {
		t.Errorf("expected at least 3 unique after merge, got %d", count)
	}
}

func TestHLL_LargeDataset(t *testing.T) {
	c := New()
	hlc := NewHyperLogLogCache(c)

	for i := 0; i < 10000; i++ {
		hlc.PFAdd("large_hll", string(rune(i)))
	}

	count, _ := hlc.PFCount("large_hll")
	errorRate := float64(count-10000) / 10000
	if errorRate < 0 {
		errorRate = -errorRate
	}
	if errorRate > 0.05 {
		t.Errorf("HLL large dataset error too high: got %d, expected ~10000, error rate %.2f%%", count, errorRate*100)
	}
}
