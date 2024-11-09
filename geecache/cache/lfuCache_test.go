package cache

import (
	"testing"
)

func TestLFUCache(t *testing.T) {
	lfu := NewLFUCache(3) // 容量为3

	// 添加三个条目
	lfu.Add("a", String("1"))
	lfu.Add("b", String("2"))
	lfu.Add("c", String("3"))

	// 访问 'a' 和 'b'，增加它们的访问频率
	lfu.Get("a")
	lfu.Get("b")

	// 添加新条目 'd'，此时容量已满，应该淘汰访问频率最低的 'c'
	lfu.Add("d", String("4"))

	if _, ok := lfu.Get("c"); ok {
		t.Error("预期 'c' 被淘汰")
	}
	if _, ok := lfu.Get("d"); !ok {
		t.Error("预期 'd' 存在于缓存中")
	}
}
