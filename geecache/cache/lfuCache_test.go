package cache_test

import (
	"GeeCache/geecache/cache"
	"fmt"
	"testing"
)

// 自定义类型，作为缓存条目的类型
type MyValue struct {
	data string
}

func (v MyValue) Len() int {
	return len(v.data) // 返回字符串的长度作为缓存条目的大小
}

func TestLFUCacheWithCustomValue(t *testing.T) {
	// k1, k2, k3 := "key1", "key2", "key3"
	// v1, v2, v3 := "value1", "value2", "value3"
	// 创建一个 LFU 缓存，最大容量为 10 字节
	lfuCache := cache.NewLFUCache(24)

	// 使用自定义类型 MyValue 添加条目
	lfuCache.Add("key1", MyValue{data: "val1"})
	lfuCache.Add("key2", MyValue{data: "val2"})
	lfuCache.Add("key3", MyValue{data: "val3"})

	// 获取缓存条目
	if value, ok := lfuCache.Get("key1"); !ok || value.Len() != 4 { // "val1" 的长度为 4
		fmt.Println(value.Len())
		t.Errorf("expected value of key1 to be 'val1', but got %v", value)
	}

	// 验证移除最旧条目（最小频率的条目）
	lfuCache.Add("key4", MyValue{data: "val4"}) // 触发移除最旧条目
	if _, ok := lfuCache.Get("key2"); ok {
		t.Errorf("expected key2 to be evicted")
	}

	// 验证其他条目
	if value, ok := lfuCache.Get("key3"); !ok || value.Len() != 4 {
		t.Errorf("expected value of key3 to be 'val3', but got %v", value)
	}

	if value, ok := lfuCache.Get("key4"); !ok || value.Len() != 4 {
		t.Errorf("expected value of key4 to be 'val4', but got %v", value)
	}
}
