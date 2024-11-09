package tests

import (
	"GeeCache/geecache/cache"
	"testing"
)

type String string

func (d String) Len() int {
	return len(d)
}

func TestCacheEvictionPolicies(t *testing.T) {
	// 测试 LRU 策略
	t.Run("LRU Eviction Policy", func(t *testing.T) {
		cache := cache.NewCache("lru", 3) // 创建 LRU 缓存，容量为 3

		cache.Add("a", String("A"))
		cache.Add("b", String("B"))
		cache.Add("c", String("C"))

		// 增加 'a' 的访问频率，使其变为最常访问的键
		cache.Get("a")

		// 添加新键 'd'，应淘汰最少访问的 'b'
		cache.Add("d", String("D"))

		if _, ok := cache.Get("b"); ok {
			t.Error("预期 'b' 被淘汰")
		}
		if _, ok := cache.Get("d"); !ok {
			t.Error("预期 'd' 存在于缓存中")
		}
	})

	// 测试 LFU 策略
	t.Run("LFU Eviction Policy", func(t *testing.T) {
		cache := cache.NewCache("lfu", 3) // 创建 LFU 缓存，容量为 3

		cache.Add("x", String("X"))
		cache.Add("y", String("Y"))
		cache.Add("z", String("Z"))

		cache.Get("x")
		cache.Get("x")
		cache.Get("y")

		// 添加新键 'w'，应淘汰访问频率最低的 'z'
		cache.Add("w", String("W"))

		if _, ok := cache.Get("z"); ok {
			t.Error("预期 'z' 被淘汰")
		}
		if _, ok := cache.Get("w"); !ok {
			t.Error("预期 'w' 存在于缓存中")
		}
	})
}
