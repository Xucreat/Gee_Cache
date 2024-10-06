package geecache

import (
	"fmt"
	pb "geecache/geecachepb"
	"geecache/singleflight"
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
// 函数类型实现某一个接口，称之为接口型函数，方便使用者在调用时既能够传入函数作为参数，
// 也能够传入实现了该接口的结构体作为参数。
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// A Group is a cache namespace and associated data loaded spread over
// 一个 Group 可以认为是一个缓存的命名空间
type Group struct {
	name      string
	getter    Getter // 缓存未命中时获取源数据的回调(callback)。
	maincache cache  // 一开始实现的并发缓存
	peers     PeerPicker
	// 使用singleflight.Group确保每个键只被获取一次
	loader *singleflight.Group
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
		loader:    &singleflight.Group{},
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
// 使用 PickPeer() 方法选择节点，若非本机节点，则调用 getFromPeer() 从远程获取。
// 若是本机节点或失败，则回退到 getLocally()。
// 能有效分散请求压力，同时保证即便远程节点出问题，系统仍然能正常工作。
func (g *Group) load(key string) (value ByteView, err error) {
	// 使用 g.loader.Do 包裹起来,确保并发场景下针对相同的 key，load 过程只会调用一次。
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		// 判断缓存系统是否有配置其他可用节点。如果没有其他节点，直接走本地获取的流程。
		if g.peers != nil {
			// 有可用的节点，则通过调用 PickPeer(key) 选择一个节点
			if peer, ok := g.peers.PickPeer(key); ok {
				// 调用 getFromPeer(peer, key) 从远程节点获取数据
				if value, err = g.getFromPeer(peer, key); err == nil {
					// 成功，则返回远程获取到的数据
					return value, nil
				}
				// 失败，则记录日志并回退到本地获取流程。
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}
		// 没有找到合适的远程节点，或者从远程节点获取数据失败，则调用 g.getLocally(key) 进行本地获取。
		return g.getLocally(key)
	})
	if err == nil {
		return viewi.(ByteView), nil
	}
	return

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

// RegisterPeers registers a PeerPicker for choosing remote peer
// 为 Group 提供了一个选择远程缓存节点的机制，之后可以通过 PeerPicker 选择合适的节点来处理缓存请求。
func (g *Group) RegisterPeers(peers PeerPicker) {
	// 对重复注册 PeerPicker 的检查，确保系统中每个 Group 只允许注册一次 peers。
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	//允许 Group 通过 PeerPicker 来挑选远程节点处理缓存请求。
	g.peers = peers
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	err := peer.Get(req, res)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
}
