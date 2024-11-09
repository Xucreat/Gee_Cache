package cache

import (
	"GeeCache/geecache/common"
	"reflect"
	"testing"
)

type String string

// 实现了Value中定义的方法，该类型实现了该接口，可以将这个类型的值赋给该接口类型的变量
func (d String) Len() int {
	return len(d)
}

func TestGet(t *testing.T) {
	lru := NewLRUCache(int64(0), nil)
	lru.Add("key1", String("1234"))
	if v, ok := lru.Get("key1"); !ok || string(v.(String)) != "1234" {
		t.Fatalf("cache hit key1=1234 failed")
	}
	// 验证键"key2"不应该在缓存中存在
	if _, ok := lru.Get("key2"); ok {
		t.Fatalf("cache miss key2 failed")
	}
}

// 测试，当使用内存超过了设定值时，是否会触发“无用”节点的移除：
func TestRemoveoldest(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "k3"
	v1, v2, v3 := "value1", "value2", "v3"
	cap := len(k1 + v1 + k2 + v2)
	lru := NewLRUCache(int64(cap), nil)
	lru.Add(k1, String(v1))
	lru.Add(k2, String(v2))
	lru.Add(k3, String(v3)) // more than cap, must to move k1,v1 out

	if _, ok := lru.Get("key1"); ok || lru.Len() != 2 {
		t.Fatal("Removeoldest key1 failed")
	}
}

// 测试回调函数能否被调用:当某些键值对被逐出缓存时，是否正确调用了回调函数并记录了被逐出的键。
func TsetOnEvicted(t *testing.T) {
	keys := make([]string, 0)
	callback := func(key string, value common.Value) {
		keys = append(keys, key)
	}
	lru := NewLRUCache(int64(10), callback)
	lru.Add("key1", String("123456"))
	lru.Add("k2", String("k2"))
	lru.Add("k3", String("k3"))
	lru.Add("k4", String("k4"))

	// 逐出一个 key1（6 个字节）虽然腾出了足够的空间，
	// 但仍然需要逐出最少数量的键值对以 确保缓存策略 的有效性。
	expect := []string{"key1", "k2"}
	if !reflect.DeepEqual(expect, keys) {
		t.Fatalf("Call OnEvicted failed, expect keys equals to %s", expect)
	}
}
