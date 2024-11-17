package tests

import (
	"GeeCache/geecache/cache"
	"GeeCache/geecache/core"
	"GeeCache/geecache/interfaces"
	"testing"
)

// String 类型实现 Len() 方法，作为缓存条目的类型
type String string

func (d String) Len() int {
	return len(d) // 返回字符串的长度作为缓存条目的大小
}

func TestCacheEvictionPolicies(t *testing.T) {
	// 测试 LRU 策略
	t.Run("LRU Eviction Policy", func(t *testing.T) {
		cache := cache.NewCache("lru", 3) // 创建 LRU 缓存，容量为 3

		// 添加缓存条目
		cache.Add("a", String("A"))
		cache.Add("b", String("B"))
		cache.Add("c", String("C"))

		// 增加 'a' 的访问频率，使其变为最常访问的键
		cache.Get("a")

		// 添加新键 'd'，应淘汰最少访问的 'b'
		cache.Add("d", String("D"))

		// 检查 'b' 是否被淘汰
		if _, ok := cache.Get("b"); ok {
			t.Error("预期 'b' 被淘汰")
		}
		// 检查 'd' 是否存在
		if _, ok := cache.Get("d"); !ok {
			t.Error("预期 'd' 存在于缓存中")
		}
	})

	// 测试 LFU 策略
	t.Run("LFU Eviction Policy", func(t *testing.T) {
		cache := cache.NewCache("lfu", 3) // 创建 LFU 缓存，容量为 3

		// 添加缓存条目
		cache.Add("x", String("X"))
		cache.Add("y", String("Y"))
		cache.Add("z", String("Z"))

		// 增加 'x' 的访问频率
		cache.Get("x")
		cache.Get("x")
		cache.Get("y")

		// 添加新键 'w'，应淘汰访问频率最低的 'z'
		cache.Add("w", String("W"))

		// 检查 'z' 是否被淘汰
		if _, ok := cache.Get("z"); ok {
			t.Error("预期 'z' 被淘汰")
		}
		// 检查 'w' 是否存在
		if _, ok := cache.Get("w"); !ok {
			t.Error("预期 'w' 存在于缓存中")
		}
	})
}

// 验证热点 Key 逻辑
func TestHotKeyDetection(t *testing.T) {
	// 模拟一个 Getter（可以是任何实现了 Getter 接口的对象）
	getter := interfaces.GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})

	// 创建一个新的缓存 Group，最大缓存大小 1024 字节
	group := core.NewGroup("test", 1024, getter, "lru") // 假设使用 LRU 算法

	// 模拟访问 Key
	for i := 0; i < 101; i++ {
		group.IncrementKeyUsage("hotkey") // 每次访问增加 hotkey 的访问次数
	}

	// 检查 'hotkey' 是否被检测为热点
	if !group.IsHotKey("hotkey") {
		t.Error("Expected hotkey to be detected as hotspot")
	}

	// 检查其他 Key 是否为冷门
	if group.IsHotKey("coldkey") {
		t.Error("Expected coldkey to not be detected as hotspot")
	}
}
