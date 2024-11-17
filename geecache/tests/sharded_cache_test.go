package tests

/*
向缓存中添加数据并确保数据正确存储。
验证缓存分片是否正常工作。
测试 Get 和 Add 方法的功能。
测试不同的缓存淘汰算法（LRU 和 LFU）是否能正确工作。
*/
import (
	"GeeCache/geecache/cache"
	"GeeCache/geecache/data"
	"strconv"
	"testing"
)

// 测试 ShardedCache 的功能
func TestShardedCache(t *testing.T) {
	// 创建一个包含 4 个分片的 ShardedCache，每个分片最大 1MB，使用 LRU 算法
	shardedCache := cache.NewShardedCache(4, 1024*1024, "lru")

	// 向分片缓存中添加数据
	numKeys := 1000
	for i := 0; i < numKeys; i++ {
		key := "key" + strconv.Itoa(i)
		value := "value" + strconv.Itoa(i)
		shardedCache.Add(key, data.ByteView{B: []byte(value)})
	}

	// 检查每个分片的缓存大小是否正确
	for i := 0; i < shardedCache.NumShards; i++ {
		shard := shardedCache.Shards[i]
		if shard.CacheBytes != 1024*1024 {
			t.Errorf("Shard %d does not have the correct capacity: expected %d, got %d", i, 1024*1024, shard.CacheBytes)
		}
	}

	// 测试缓存是否能正确读取数据
	for i := 0; i < numKeys; i++ {
		key := "key" + strconv.Itoa(i)
		expectedValue := "value" + strconv.Itoa(i)

		value, ok := shardedCache.Get(key)
		if !ok {
			t.Errorf("Failed to retrieve value for key %s", key)
		} else if string(value.(data.ByteView).ByteSlice()) != expectedValue {
			t.Errorf("Expected value for key %s is %s, got %s", key, expectedValue, string(value.(data.ByteView).ByteSlice()))
		}
	}
}

// 测试不同算法下的缓存添加和获取
func TestCacheAlgorithm(t *testing.T) {
	// 测试 LRU 算法
	lruCache := cache.NewConcurrentCache(1024*1024, "lru")
	testCacheAlgorithm(t, lruCache)

	// 测试 LFU 算法
	lfuCache := cache.NewConcurrentCache(1024*1024, "lfu")
	testCacheAlgorithm(t, lfuCache)
}

// 验证不同缓存淘汰算法下的缓存操作
func testCacheAlgorithm(t *testing.T, cache *cache.ConcurrentCache) {
	// 向缓存中添加数据
	key := "key1"
	value := "value1"
	cache.Add(key, data.ByteView{B: []byte(value)})

	// 获取缓存中的数据并验证
	val, ok := cache.Get(key)
	if !ok {
		t.Errorf("Failed to get value for key %s", key)
	} else if string(val.(data.ByteView).ByteSlice()) != value {
		t.Errorf("Expected value for key %s is %s, got %s", key, value, string(val.(data.ByteView).ByteSlice()))
	}

	// 添加更多数据以触发淘汰机制
	for i := 2; i <= 100; i++ {
		cache.Add("key"+strconv.Itoa(i), data.ByteView{B: []byte("value" + strconv.Itoa(i))})
	}

	// 确保最初添加的缓存项仍然存在（这部分验证可以根据缓存算法的具体实现来调整）
	val, ok = cache.Get(key)
	if !ok {
		t.Errorf("Cache evicted item that should not have been evicted (key: %s)", key)
	} else if string(val.(data.ByteView).ByteSlice()) != value {
		t.Errorf("Expected value for key %s is %s, got %s", key, value, string(val.(data.ByteView).ByteSlice()))
	}
}

// 测试分片缓存的 `getShard` 方法
func TestGetShard(t *testing.T) {
	shardedCache := cache.NewShardedCache(4, 1024*1024, "lru")

	// 验证 `getShard` 方法
	key := "key1"
	shard := shardedCache.GetShard(key)

	// 验证返回的 shard 是否正确
	if shard == nil {
		t.Errorf("getShard failed for key %s", key)
	}

	// 确保每次返回的 shard 都是合法的
	expectedShardID := shardedCache.GetShardID(key)
	if shard != shardedCache.Shards[expectedShardID] {
		t.Errorf("getShard returned incorrect shard for key %s", key)
	}
}

// 测试分片缓存的容量
func TestShardedCacheCapacity(t *testing.T) {
	shardedCache := cache.NewShardedCache(4, 1024*1024, "lfu")

	// 向缓存中添加数据
	numKeys := 1000
	for i := 0; i < numKeys; i++ {
		key := "key" + strconv.Itoa(i)
		value := "value" + strconv.Itoa(i)
		shardedCache.Add(key, data.ByteView{B: []byte(value)})
	}

	// 检查每个分片的缓存大小是否符合预期
	for i := 0; i < shardedCache.NumShards; i++ {
		shard := shardedCache.Shards[i]
		if shard.CacheBytes != 1024*1024 {
			t.Errorf("Shard %d exceeded its capacity: expected %d, got %d", i, 1024*1024, shard.CacheBytes)
		}
	}
}
