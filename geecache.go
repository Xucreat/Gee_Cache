package geecache

import (
	"fmt"
	"log"
	"sync"
)

// A Getter loads data for a key
type Getter interface {
	Get(key string) ([]byte, error)
}

// A Getter implements Getter with a function
type GetterFunc func(key string) ([]byte, error)

// Get implements Getter interface function
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// A Group is a cache namespace and associated data loaded spread over
// 一个 Group 可以认为是一个缓存的命名空间
type Group struct {
	name      string
	getter    Getter // 缓存未命中时获取源数据的回调(callback)。
	maincache cache  // 一开始实现的并发缓存
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group) // 将 group 存储在全局变量 groups 中
)

// NewGroup create a new instance of Group
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		maincache: cache{cacheBytes: cacheBytes},
	}
	groups[name] = g
	return g
}

// GetGroup returns the named group previously created with NewGroup
// or nil if there is no such group
func GetGroup(name string) *Group {
	// if groups[name] != nil {
	// 	return groups[name]
	// }else {return nil}

	// 使用了只读锁 RLock()，因为不涉及任何冲突变量的写操作。
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// value for key from cache
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	//流程 ⑴ ：从 mainCache 中查找缓存，如果存在则返回缓存值。
	/*调用 get() 时，不需要复制，ByteView 是只读的，不可修改。
	通过 ByteSlice() 或 String() 方法取到缓存值的副本。
	只读属性，是设计 ByteView 的主要目的之一。*/
	if v, ok := g.maincache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}

	// 流程 ⑶ ：缓存不存在，则调用 load 方法，
	return g.load(key)
}

// load 调用 getLocally（分布式场景下会调用 getFromPeer 从其他节点获取），
func (g *Group) load(key string) (value ByteView, err error) {
	return g.getLocally(key)
}

// getLocally 调用用户回调函数 g.getter.Get() 获取源数据，
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}

	// 将源数据添加到缓存 mainCache 中（通过 populateCache 方法）
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value ByteView) {
	g.maincache.add(key, value)
}
