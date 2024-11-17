package cache

import (
	"GeeCache/geecache/common"
	"GeeCache/geecache/data"
	"GeeCache/geecache/interfaces"
	"hash/fnv"

	"sync"
)

type Cache interface {
	Get(key string) (common.Value, bool)
	Add(key string, value common.Value)
	Len() int
}

// ConcurrentCache 结构体定义带并发特性的 LRU 缓存
type ConcurrentCache struct {
	mu         sync.Mutex // 互斥锁
	cache      interfaces.EvictionPolicy
	CacheBytes int64  // 字段名首字母大写表示它是导出的，可以在其他包中访问
	Algorithm  string // 新增字段，用于指定算法类型
}

// ShardedCache 分片缓存，提升并发性能, 一个包含多个 ConcurrentCache 实例的缓存系统
type ShardedCache struct {
	Shards    []*ConcurrentCache
	NumShards int
}

// NewConcurrentCache 创建一个并发缓存，支持动态选择算法
func NewConcurrentCache(cacheBytes int64, algorithm string) *ConcurrentCache {
	var cache interfaces.EvictionPolicy
	switch algorithm {
	case "lru":
		cache = NewLRUCache(cacheBytes, nil)
	case "lfu":
		cache = NewLFUCache(int(cacheBytes))
	default:
		panic("unsupported cache algorithm")
	}
	return &ConcurrentCache{
		CacheBytes: cacheBytes,
		cache:      cache,
		Algorithm:  algorithm,
	}
}

// 将缓存分片（sharding），ShardedCache 可以提高并发性能，因为不同的 Goroutine 可以访问不同的缓存分片，避免了全局锁竞争。
func NewShardedCache(numShards int, cacheBytes int64, algorithm string) *ShardedCache {
	// shards 切片保存每个分片的 ConcurrentCache 实例。
	shards := make([]*ConcurrentCache, numShards)
	for i := 0; i < numShards; i++ {
		shards[i] = NewConcurrentCache(cacheBytes, algorithm)
	}
	return &ShardedCache{
		Shards:    shards,
		NumShards: numShards,
	}
}

// getShard 根据键计算对应的分片
func (s *ShardedCache) GetShard(key string) *ConcurrentCache {
	h := fnv.New32a()
	h.Write([]byte(key))
	return s.Shards[int(h.Sum32())%s.NumShards]
}

// Add 向缓存中添加数据 !!!
func (c *ConcurrentCache) Add(key string, value common.Value) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache.Add(key, value)

	// // 延迟初始化->该对象的创建将会延迟至第一次使用该对象时
	// // 提高性能，并减少程序内存要求
	// if c.Lru == nil {
	// 	c.Lru = NewLRUCache(c.CacheBytes, nil)
	// }
	// c.Lru.Add(key, value)

}

func (c *ConcurrentCache) Get(key string) (common.Value, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cache.Get(key)
}

// Add 向分片缓存中添加数据
func (s *ShardedCache) Add(key string, value data.ByteView) {
	shard := s.GetShard(key)
	shard.Add(key, value)
}

func (s *ShardedCache) Get(key string) (common.Value, bool) {
	shard := s.GetShard(key)
	return shard.Get(key)
}

// GetShardID 根据键计算对应的分片 ID
func (s *ShardedCache) GetShardID(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32()) % s.NumShards
}
