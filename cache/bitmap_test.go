package cache

import (
	"testing"
)

func TestBitmap_SetAndGet(t *testing.T) {
	c := New()
	bc := NewBitmapCache(c)

	old, err := bc.SetBit("mykey", 7, 1)
	if err != nil {
		t.Fatalf("SetBit failed: %v", err)
	}
	if old != 0 {
		t.Errorf("expected old bit 0, got %d", old)
	}

	bit, err := bc.GetBit("mykey", 7)
	if err != nil {
		t.Fatalf("GetBit failed: %v", err)
	}
	if bit != 1 {
		t.Errorf("expected bit 1, got %d", bit)
	}

	bit, err = bc.GetBit("mykey", 0)
	if err != nil {
		t.Fatalf("GetBit failed: %v", err)
	}
	if bit != 0 {
		t.Errorf("expected bit 0 for unset position, got %d", bit)
	}
}

func TestBitmap_SetBitOverwrite(t *testing.T) {
	c := New()
	bc := NewBitmapCache(c)

	bc.SetBit("key", 3, 1)
	old, err := bc.SetBit("key", 3, 0)
	if err != nil {
		t.Fatalf("SetBit failed: %v", err)
	}
	if old != 1 {
		t.Errorf("expected old bit 1, got %d", old)
	}

	bit, _ := bc.GetBit("key", 3)
	if bit != 0 {
		t.Errorf("expected bit 0 after clearing, got %d", bit)
	}
}

func TestBitmap_GetBitNonExistent(t *testing.T) {
	c := New()
	bc := NewBitmapCache(c)

	bit, err := bc.GetBit("nonexistent", 5)
	if err != nil {
		t.Fatalf("GetBit failed: %v", err)
	}
	if bit != 0 {
		t.Errorf("expected 0 for non-existent key, got %d", bit)
	}
}

func TestBitmap_BitCount(t *testing.T) {
	c := New()
	bc := NewBitmapCache(c)

	bc.SetBit("key", 0, 1)
	bc.SetBit("key", 3, 1)
	bc.SetBit("key", 7, 1)

	count, err := bc.BitCountAll("key")
	if err != nil {
		t.Fatalf("BitCountAll failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 set bits, got %d", count)
	}
}

func TestBitmap_BitCountRange(t *testing.T) {
	c := New()
	bc := NewBitmapCache(c)

	bc.SetBit("key", 0, 1)
	bc.SetBit("key", 9, 1)
	bc.SetBit("key", 15, 1)

	count, err := bc.BitCount("key", 0, 0)
	if err != nil {
		t.Fatalf("BitCount failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 set bit in byte 0, got %d", count)
	}

	count, err = bc.BitCount("key", 1, 1)
	if err != nil {
		t.Fatalf("BitCount failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 set bits in byte 1, got %d", count)
	}
}

func TestBitmap_BitCountNonExistent(t *testing.T) {
	c := New()
	bc := NewBitmapCache(c)

	count, err := bc.BitCountAll("nonexistent")
	if err != nil {
		t.Fatalf("BitCountAll failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 for non-existent key, got %d", count)
	}
}

func TestBitmap_BitOpAnd(t *testing.T) {
	c := New()
	bc := NewBitmapCache(c)

	bc.SetBit("src1", 0, 1)
	bc.SetBit("src1", 1, 1)
	bc.SetBit("src2", 0, 1)
	bc.SetBit("src2", 2, 1)

	result, err := bc.BitOp("AND", "dest", "src1", "src2")
	if err != nil {
		t.Fatalf("BitOp AND failed: %v", err)
	}
	if result <= 0 {
		t.Errorf("expected positive result, got %d", result)
	}

	bit0, _ := bc.GetBit("dest", 0)
	bit1, _ := bc.GetBit("dest", 1)
	bit2, _ := bc.GetBit("dest", 2)

	if bit0 != 1 {
		t.Errorf("expected dest[0]=1, got %d", bit0)
	}
	if bit1 != 0 {
		t.Errorf("expected dest[1]=0, got %d", bit1)
	}
	if bit2 != 0 {
		t.Errorf("expected dest[2]=0, got %d", bit2)
	}
}

func TestBitmap_BitOpOr(t *testing.T) {
	c := New()
	bc := NewBitmapCache(c)

	bc.SetBit("src1", 0, 1)
	bc.SetBit("src2", 1, 1)

	bc.BitOp("OR", "dest", "src1", "src2")

	bit0, _ := bc.GetBit("dest", 0)
	bit1, _ := bc.GetBit("dest", 1)

	if bit0 != 1 || bit1 != 1 {
		t.Errorf("OR: expected dest[0]=1, dest[1]=1, got %d, %d", bit0, bit1)
	}
}

func TestBitmap_BitOpXor(t *testing.T) {
	c := New()
	bc := NewBitmapCache(c)

	bc.SetBit("src1", 0, 1)
	bc.SetBit("src1", 1, 1)
	bc.SetBit("src2", 0, 1)

	bc.BitOp("XOR", "dest", "src1", "src2")

	bit0, _ := bc.GetBit("dest", 0)
	bit1, _ := bc.GetBit("dest", 1)

	if bit0 != 0 {
		t.Errorf("XOR: expected dest[0]=0, got %d", bit0)
	}
	if bit1 != 1 {
		t.Errorf("XOR: expected dest[1]=1, got %d", bit1)
	}
}

func TestBitmap_BitOpNot(t *testing.T) {
	c := New()
	bc := NewBitmapCache(c)

	bc.SetBit("src", 0, 1)

	bc.BitOp("NOT", "dest", "src")

	bit0, _ := bc.GetBit("dest", 0)
	bit1, _ := bc.GetBit("dest", 1)

	if bit0 != 0 {
		t.Errorf("NOT: expected dest[0]=0, got %d", bit0)
	}
	if bit1 != 1 {
		t.Errorf("NOT: expected dest[1]=1, got %d", bit1)
	}
}

func TestBitmap_BitPos(t *testing.T) {
	c := New()
	bc := NewBitmapCache(c)

	bc.SetBit("key", 3, 1)
	bc.SetBit("key", 10, 1)

	pos, err := bc.BitPos("key", 1, 0, -1, true)
	if err != nil {
		t.Fatalf("BitPos failed: %v", err)
	}
	if pos != 3 {
		t.Errorf("expected first set bit at position 3, got %d", pos)
	}
}

func TestBitmap_BitPosZero(t *testing.T) {
	c := New()
	bc := NewBitmapCache(c)

	bc.SetBit("key", 0, 1)
	bc.SetBit("key", 1, 1)

	pos, err := bc.BitPos("key", 0, 0, -1, true)
	if err != nil {
		t.Fatalf("BitPos failed: %v", err)
	}
	if pos != 2 {
		t.Errorf("expected first zero bit at position 2, got %d", pos)
	}
}

func TestBitmap_NegativeOffset(t *testing.T) {
	c := New()
	bc := NewBitmapCache(c)

	_, err := bc.SetBit("key", -1, 1)
	if err == nil {
		t.Error("expected error for negative offset")
	}
}

func TestBitmap_HighOffset(t *testing.T) {
	c := New()
	bc := NewBitmapCache(c)

	old, err := bc.SetBit("key", 1000000, 1)
	if err != nil {
		t.Fatalf("SetBit with high offset failed: %v", err)
	}
	if old != 0 {
		t.Errorf("expected old bit 0, got %d", old)
	}

	bit, _ := bc.GetBit("key", 1000000)
	if bit != 1 {
		t.Errorf("expected bit 1 at high offset, got %d", bit)
	}
}
