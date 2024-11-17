package cache

import (
	"GeeCache/geecache/common"
	"container/list"
)

// var _ cache.EvictionPolicy = (*Cache)(nil)

// Cache is a LRU cache.It is not safe for concurrent access
type LRUCache struct {
	maxBytes int64                    // max memory allowed to be used
	nbytes   int64                    // used memory
	ll       *list.List               // 使用 Go 语言标准库实现双向链表list.List。
	Cache    map[string]*list.Element // 值是双向链表中对应节点的指针

	// 某条记录被移除时的回调函数. 可为 nil
	// optional and executed when an entry is purged.
	OnEvicted func(key string, value common.Value)
}

// 双向链表节点的数据类型
type entry struct {
	key   string // 在链表中仍保存每个值对应的 key 的好处:淘汰队首节点时，需要用 key 从字典中删除对应的映射。
	value common.Value
}

// // Value use Len to count how many bytes it takes
// type Value interface {
// 	Len() int
// }

func NewLRUCache(maxBytes int64, onEvicted func(string, common.Value)) *LRUCache {
	return &LRUCache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		Cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// Search Function
// 1.Get look ups a key's value
func (c *LRUCache) Get(key string) (value common.Value, ok bool) {
	if ele, ok := c.Cache[key]; ok { // ele is 双向链表中对应节点的指针
		c.ll.MoveToFront(ele) // 将链表中的节点 ele 移动到队尾（双向链表作为队列，队首队尾是相对的，在这里约定 front 为队尾）

		// ele.Value 是一个接口类型，这里假设实际存储的值是 *entry 类型。
		// (*entry) 是一个类型断言，用于将 ele.Value 转换为具体的 *entry 类型。
		// 如果 ele.Value 实际上是 *entry 类型的值，类型断言将成功，并返回这个值；否则，程序会发生运行时错误（panic）。
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

// RemoveOldest removes the oldest item
func (c *LRUCache) RemoveOldest() {
	ele := c.ll.Back() // 取到队首节点，从链表中删除
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.Cache, kv.key)                                // 从字典 c.Cache 中删除该节点的映射关系
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len()) // 更新当前所用的内存 c.nbytes
		// 回调函数 OnEvicted 不为 nil，则调用回调函数
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// 3.Add adds a value to the Cache.
func (c *LRUCache) Add(key string, value common.Value) {
	if ele, ok := c.Cache[key]; ok { // 键存在，则更新对应节点的值，并将该节点移到队尾。
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len()) // new-old
		kv.value = value
	} else { // 不存在则
		ele := c.ll.PushFront(&entry{key, value}) // 队尾添加新节点 &entry{key, value}
		c.Cache[key] = ele                        // 字典中添加 key 和节点的映射关系
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes != 0 && c.maxBytes < c.nbytes { // 超过了设定的最大值 c.maxBytes
		c.RemoveOldest()
	}
}

// Len the number of Cache entries
func (c *LRUCache) Len() int {
	return c.ll.Len()
}
