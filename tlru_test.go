package tlru

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTLRU(t *testing.T) {
	t.Run("OverrideValue", func(t *testing.T) {
		c := New[string](ConstantCost[int], 10)
		c.Set("a", 10, time.Second)
		c.Set("a", 20, time.Second)
		v, deadline, ok := c.Get("a")
		if !ok {
			t.Fatalf("entry doesn't exist")
		}
		if v != 20 {
			t.Fatalf("v is %v", v)
		}
		require.WithinDuration(t, deadline, time.Now().Add(time.Second), time.Millisecond)
	})
	t.Run("OldValuesEvicted", func(t *testing.T) {
		c := New[string](ConstantCost[int], 10)
		for i := 0; i < 100; i++ {
			c.Set(strconv.Itoa(i), i, time.Second)
			// 4 is our busy value that should not be evicted.
			c.Get("4")
		}
		for i := 0; i < 100; i++ {
			v, _, ok := c.Get(strconv.Itoa(i))
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
		c := New[string](ConstantCost[int], 10)
		c.Set("a", 10, time.Second)
		c.Delete("a")
		if c.cost != 0 {
			t.Fatalf("cost is %v", c.cost)
		}
		v, _, ok := c.Get("a")
		if ok {
			t.Fatalf("value %v:%v still exists", "a", v)
		}
	})

	t.Run("DynamicCost", func(t *testing.T) {
		c := New[string](
			func(v string) int {
				return len(v)
			},
			100,
		)
		c.Set("some_key", "some value", time.Second)
	})

	t.Run("UnlimitedCost", func(t *testing.T) {
		c := New[int, int](nil, -1)

		for i := 0; i < 100; i++ {
			c.Set(i, i+1, time.Second)
		}

		for i := 0; i < 100; i++ {
			ii, _, ok := c.Get(i)
			require.True(t, ok)
			require.Equal(t, i+1, ii)
		}
	})

	t.Run("Do", func(t *testing.T) {
		c := New[string, int](nil, -1)

		n := 10
		fn := func() (int, error) {
			n += 1
			return n, nil
		}

		v, err := c.Do("a", fn, time.Second)
		require.NoError(t, err)

		require.Equal(t, 11, v)

		v, err = c.Do("a", fn, time.Second)
		require.NoError(t, err)

		// No recompute, cache hit.
		require.Equal(t, 11, v)
	})
}

func TestTLRU_Expires(t *testing.T) {
	t.Parallel()
	t.Run("ImmediateExpirey", func(t *testing.T) {
		t.Parallel()
		c := New[string](ConstantCost[int], 10)
		// This entry should immediately expire.
		c.Set("a", 10, 0)
		_, _, ok := c.Get("a")
		require.False(t, ok)
	})
	t.Run("NeverExpires", func(t *testing.T) {
		t.Parallel()
		c := New[string](ConstantCost[int], 10)
		c.Set("a", 10, time.Hour*999)
		time.Sleep(time.Second)
		_, _, ok := c.Get("a")
		require.True(t, ok)
	})
}

func Benchmark_TLRU_Get(b *testing.B) {
	c := New[string](ConstantCost[int], 1000)
	c.Set("test-key", 10, time.Second)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Get("test-key")
	}
}

func Benchmark_TLRU_Set(b *testing.B) {
	c := New[string](ConstantCost[int], 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Set("test-key-"+strconv.Itoa(i), 10, time.Second)
	}
}
