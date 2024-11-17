package cache

import (
	"GeeCache/geecache/common"
	"GeeCache/geecache/data"
	"container/list"
	"fmt"
	"log"
)

// LFUCache 是一个 LFU 缓存，非并发安全。
type LFUCache struct {
	maxBytes int                    // 最大容量
	nbytes   int                    // 当前缓存大小
	minFreq  int                    // 最小频率
	freqMap  map[int]*list.List     // 频率到条目列表的映射
	cache    map[string]*data.Entry // 键到条目的映射
}

// NewLFUCache 创建一个 LFU 缓存实例
func NewLFUCache(maxBytes int) *LFUCache {
	return &LFUCache{
		maxBytes: maxBytes,
		nbytes:   0, // 确保 nbytes 从 0 开始
		freqMap:  make(map[int]*list.List),
		cache:    make(map[string]*data.Entry),
	}
}

// Get 获取缓存条目
func (c *LFUCache) Get(key string) (value common.Value, ok bool) {
	if entry, ok := c.cache[key]; ok {
		// 调用增频函数
		c.incrementFrequency(entry)
		log.Printf("Cache hit for key: %s, frequency: %d", key, entry.Frequency)
		return entry.Value, true
	}
	log.Printf("Cache miss for key: %s", key)
	return nil, false
}

// Add 添加或更新缓存条目
func (c *LFUCache) Add(key string, value common.Value) {
	fmt.Printf("Adding key: %s with value: %v, current nbytes: %d\n", key, value, c.nbytes)
	if entry, ok := c.cache[key]; ok {
		// 如果缓存中已存在，更新条目的值并增加频率
		entry.Value = value
		c.incrementFrequency(entry) // 更新频率
		c.nbytes += value.Len() - entry.Value.Len()
		log.Printf("Updated key: %s, new frequency: %d", key, entry.Frequency)
	} else {
		// 新增条目
		if c.nbytes+value.Len() > c.maxBytes {
			fmt.Println(c.nbytes)
			c.RemoveOldest() // 超出容量时移除最旧条目
		}
		newEntry := &data.Entry{
			Key:       key,
			Value:     value,
			Frequency: 1, // 新条目的频率为 1
		}
		c.cache[key] = newEntry // 将新条目添加到 cache
		if _, ok := c.freqMap[1]; !ok {
			c.freqMap[1] = list.New() // 创建频率 1 的链表
		}
		c.freqMap[1].PushBack(newEntry)    // 将条目添加到频率 1 的链表中
		c.minFreq = 1                      // 确保最小频率为 1
		c.nbytes += len(key) + value.Len() // 更新缓存的大小
		log.Printf("Added new key: %s, frequency: %d", key, newEntry.Frequency)
	}
}

// RemoveOldest 移除频率最低的条目
func (c *LFUCache) RemoveOldest() {
	if freqList, ok := c.freqMap[c.minFreq]; ok && freqList.Len() > 0 {
		oldest := freqList.Front()
		if oldest != nil {
			entry := oldest.Value.(*data.Entry)
			// 从缓存中移除条目
			delete(c.cache, entry.Key)
			// 从频率列表中移除条目
			freqList.Remove(oldest)
			c.nbytes -= len(entry.Key) + entry.Value.Len()
			// 如果频率列表为空，删除频率列表并更新最小频率
			if freqList.Len() == 0 {
				delete(c.freqMap, c.minFreq)
				// 更新 minFreq
				for newMinFreq := c.minFreq + 1; newMinFreq <= len(c.freqMap); newMinFreq++ {
					if _, exists := c.freqMap[newMinFreq]; exists {
						c.minFreq = newMinFreq
						break
					}
				}
			}
			log.Printf("Removed oldest key: %s, frequency: %d", entry.Key, c.minFreq)
		}
	}
}

// incrementFrequency 增加条目的访问频率
func (c *LFUCache) incrementFrequency(entry *data.Entry) {
	// 获取当前频率列表
	freqList := c.freqMap[entry.Frequency]
	// 从当前频率列表中移除条目
	for elem := freqList.Front(); elem != nil; elem = elem.Next() {
		if elem.Value.(*data.Entry) == entry {
			freqList.Remove(elem)
			break
		}
	}
	// 如果当前频率列表为空，删除频率列表并更新最小频率
	if freqList.Len() == 0 {
		delete(c.freqMap, entry.Frequency)
		// 如果最小频率被移除，更新 minFreq
		if c.minFreq == entry.Frequency {
			c.minFreq++
		}
	}

	// 增加频率并添加到新的频率列表
	entry.Frequency++
	if _, ok := c.freqMap[entry.Frequency]; !ok {
		c.freqMap[entry.Frequency] = list.New()
	}
	c.freqMap[entry.Frequency].PushBack(entry)
}

// // addEntry 添加条目到频率列表
// func (c *LFUCache) addEntry(e *data.Entry) {
// 	if _, ok := c.freqMap[e.Frequency]; !ok {
// 		c.freqMap[e.Frequency] = list.New()
// 	}
// 	c.freqMap[e.Frequency].PushFront(e)
// 	if e.Frequency == 1 || e.Frequency < c.minFreq {
// 		c.minFreq = e.Frequency
// 	}
// }
// // RemoveOldest 移除频率最低的条目
// func (c *LFUCache) RemoveOldest() {
// 	if freqList, ok := c.freqMap[c.minFreq]; ok {
// 		oldest := freqList.Front()
// 		if oldest != nil {
// 			entry := oldest.Value.(*data.Entry)
// 			delete(c.cache, entry.Key)
// 			freqList.Remove(oldest)
// 			c.nbytes -= len(entry.Key) + len(entry.Value.B)
// 			if freqList.Len() == 0 {
// 				delete(c.freqMap, c.minFreq)
// 			}
// 		}
// 	}
// }

// // removeLeastFrequent 移除最少使用的条目
// func (c *LFUCache) removeLeastFrequent() {
// 	if freqList, ok := c.freqMap[c.minFreq]; ok {
// 		leastElem := freqList.Back()
// 		if leastElem != nil {
// 			leastEntry := leastElem.Value.(*data.Entry)
// 			freqList.Remove(leastElem)
// 			delete(c.cache, leastEntry.Key)
// 			c.nbytes--
// 		}
// 	}
// }

// Len 返回缓存条目数
func (c *LFUCache) Len() int {
	return len(c.cache)
}
