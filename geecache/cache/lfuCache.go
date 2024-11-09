package cache

import (
	"GeeCache/geecache/common"
	"GeeCache/geecache/data"
	"container/list"
)

// LFUCache 是一个 LFU 缓存，非并发安全。
type LFUCache struct {
	capacity int                    // 最大容量
	size     int                    // 当前缓存大小
	minFreq  int                    // 最小频率
	freqMap  map[int]*list.List     // 频率到条目列表的映射
	keyMap   map[string]*data.Entry // 键到条目的映射
}

// NewLFUCache 创建一个 LFU 缓存实例
func NewLFUCache(capacity int) *LFUCache {
	return &LFUCache{
		capacity: capacity,
		freqMap:  make(map[int]*list.List),
		keyMap:   make(map[string]*data.Entry),
	}
}

// Get 获取值并增加条目频率, 如果键存在缓存中
func (c *LFUCache) Get(key string) (common.Value, bool) {
	if e, ok := c.keyMap[key]; ok {
		c.removeEntry(e)
		e.Frequency++
		c.addEntry(e)
		return e.Value, true
	}
	return nil, false
}

// Add 添加值到缓存
func (c *LFUCache) Add(key string, value common.Value) {
	if c.capacity <= 0 {
		return
	}

	if e, ok := c.keyMap[key]; ok {
		e.Value = value
		c.incrementFrequency(e)
	} else {
		if c.size >= c.capacity {
			c.removeLeastFrequent()
		}
		e := &data.Entry{Key: key, Value: value, Frequency: 1}
		c.keyMap[key] = e
		c.addEntry(e)
		c.size++
	}
}

// incrementFrequency 增加条目频率
func (c *LFUCache) incrementFrequency(e *data.Entry) {
	c.removeEntry(e)
	e.Frequency++
	c.addEntry(e)
}

// addEntry 添加条目到频率列表
func (c *LFUCache) addEntry(e *data.Entry) {
	if _, ok := c.freqMap[e.Frequency]; !ok {
		c.freqMap[e.Frequency] = list.New()
	}
	c.freqMap[e.Frequency].PushFront(e)
	if e.Frequency == 1 || e.Frequency < c.minFreq {
		c.minFreq = e.Frequency
	}
}

// removeEntry 从频率列表中删除条目
func (c *LFUCache) removeEntry(e *data.Entry) {
	freqList := c.freqMap[e.Frequency]
	for elem := freqList.Front(); elem != nil; elem = elem.Next() {
		if elem.Value.(*data.Entry) == e {
			freqList.Remove(elem)
			break
		}
	}
	if freqList.Len() == 0 {
		delete(c.freqMap, e.Frequency)
		if c.minFreq == e.Frequency {
			c.minFreq++
		}
	}
}

// removeLeastFrequent 移除最少使用的条目
func (c *LFUCache) removeLeastFrequent() {
	if freqList, ok := c.freqMap[c.minFreq]; ok {
		leastElem := freqList.Back()
		if leastElem != nil {
			leastEntry := leastElem.Value.(*data.Entry)
			freqList.Remove(leastElem)
			delete(c.keyMap, leastEntry.Key)
			c.size--
		}
	}
}

// Len 返回缓存条目数
func (c *LFUCache) Len() int {
	return c.size
}
