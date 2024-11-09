package cache

/*1
为 lru.Cache 添加并发特性
实例化 lru，封装 get 和 add 方法，并添加互斥锁 mu。
*/
import (
	"GeeCache/geecache/common"
	"GeeCache/geecache/data"

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
	lru        *LRUCache
	cacheBytes int64
}

// NewConcurrentCache 创建一个新的 ConcurrentCache 实例
func NewConcurrentCache(cacheBytes int64) *ConcurrentCache {
	return &ConcurrentCache{
		cacheBytes: cacheBytes,
	}
}

func (c *ConcurrentCache) Add(key string, value data.ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 延迟初始化->该对象的创建将会延迟至第一次使用该对象时
	// 提高性能，并减少程序内存要求
	if c.lru == nil {
		c.lru = NewLRUCache(c.cacheBytes, nil)
	}
	c.lru.Add(key, value)

}

func (c *ConcurrentCache) Get(key string) (value data.ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return
	}
	if v, ok := c.lru.Get(key); ok {
		return v.(data.ByteView), ok // v.(ByteView):类型断言
	}
	return
}
