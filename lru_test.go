package cache

import (
	"strconv"
	"testing"
)

func TestLRU(t *testing.T) {
	t.Run("OverrideValue", func(t *testing.T) {
		c := NewLRU[int](ConstantCost[int], 10)
		c.Set("a", 10)
		c.Set("a", 20)
		v, ok := c.Get("a")
		if !ok {
			t.Fatalf("value doesn't exist")
		}
		if v != 20 {
			t.Fatalf("v is %v", v)
		}
	})
	t.Run("OldValuesEvicted", func(t *testing.T) {
		c := NewLRU[int](ConstantCost[int], 10)
		for i := 0; i < 100; i++ {
			c.Set(strconv.Itoa(i), i)
			// 4 is our busy value that should not be evicted.
			c.Get("4")
		}
		for i := 0; i < 100; i++ {
			v, ok := c.Get(strconv.Itoa(i))
			if i < 91 && i != 4 {
				if ok {
					t.Fatalf("value %v:%v exists", i, v)
				}
				continue
			}
			if !ok {
				t.Fatalf("value %v:%v should be in cache", i, v)
			}
			if c.cost != 10 {
				t.Fatalf("cost is %v", c.cost)
			}
			if len(c.index) != 10 {
				t.Fatalf("len(c.index) is %v", len(c.index))
			}
		}
	})
	t.Run("DeleteEntry", func(t *testing.T) {
		c := NewLRU[int](ConstantCost[int], 10)
		c.Set("a", 10)
		c.Delete("a")
		if c.cost != 0 {
			t.Fatalf("cost is %v", c.cost)
		}
		v, ok := c.Get("a")
		if ok {
			t.Fatalf("value %v:%v still exists", "a", v)
		}
	})
}
