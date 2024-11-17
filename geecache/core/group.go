package core

import (
	"GeeCache/geecache/cache"
	"GeeCache/geecache/data"
	pb "GeeCache/geecache/geecachepb"
	"GeeCache/geecache/interfaces"
	"fmt"
	"log"
	"sync"
)

// A Group is a cache namespace and associated data loaded spread over
// 一个 Group 可以认为是一个缓存的命名空间
type Group struct {
	name      string
	getter    interfaces.Getter   // 缓存未命中时获取源数据的回调(callback)。
	maincache *cache.ShardedCache // 一开始实现的并发缓存
	peers     interfaces.PeerPicker
	// 使用singleflight.Group确保每个键只被获取一次
	loader      *RequestGroup
	hotKeys     map[string]int // 热点 Key 的访问统计
	hotKeyMutex sync.RWMutex   // 使用读写锁     // 防止并发访问冲突
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group) // 将 group 存储在全局变量 groups 中
)

// NewGroup create a new instance of Group
func NewGroup(name string, cacheBytes int64, getter interfaces.Getter, algorithm string) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()

	// 创建一个带有分片的缓存，支持不同的缓存算法（LRU、LFU等）
	mainCache := cache.NewShardedCache(256, cacheBytes, algorithm)

	// 创建新的 Group 对象
	g := &Group{
		name:      name,
		getter:    getter,
		maincache: mainCache, // 使用传入的分片缓存
		loader:    &RequestGroup{},
		hotKeys:   make(map[string]int), // 初始化热点统计
	}
	// 将新创建的 Group 注册到 groups 中
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
func (g *Group) Get(key string) (data.ByteView, error) {
	if key == "" {
		return data.ByteView{}, fmt.Errorf("key is required")
	}
	g.IncrementKeyUsage(key) // 增加访问计数

	//流程 ⑴ ：从 mainCache 中查找缓存，如果存在则返回缓存值。
	/*调用 get() 时，不需要复制，core.ByteView 是只读的，不可修改。
	通过 ByteSlice() 或 String() 方法取到缓存值的副本。
	只读属性，是设计 core.ByteView 的主要目的之一。*/
	if v, ok := g.maincache.Get(key); ok {
		log.Printf("[GeeCache] Cache hit for key: %s", key)
		return v.(data.ByteView), nil
	}

	log.Printf("[GeeCache] Cache miss for key: %s, loading...", key)
	// 流程 ⑶ ：缓存不存在，则调用 load 方法，
	return g.load(key)
}

// load 调用 getLocally（分布式场景下会调用 getFromPeer 从其他节点获取），
// 使用 PickPeer() 方法选择节点，若非本机节点，则调用 getFromPeer() 从远程获取。
// 若是本机节点或失败，则回退到 getLocally()。
// 能有效分散请求压力，同时保证即便远程节点出问题，系统仍然能正常工作。
func (g *Group) load(key string) (value data.ByteView, err error) {
	// 使用 g.loader.Do 包裹起来,确保并发场景下针对相同的 key，load 过程只会调用一次。
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		// 判断缓存系统是否有配置其他可用节点。如果没有其他节点，直接走本地获取的流程。
		if g.peers != nil {
			// 有可用的节点，则通过调用 PickPeer(key) 选择一个节点
			if peer, ok := g.peers.PickPeer(key); ok {
				log.Printf("[GeeCache] Trying to load key: %s from peer", key)
				// 调用 getFromPeer(peer, key) 从远程节点获取数据
				if value, err = g.getFromPeer(peer, key); err == nil {
					// 成功，则返回远程获取到的数据
					return value, nil
				}
				// 失败，则记录日志并回退到本地获取流程。
				log.Printf("[GeeCache] Failed to load key: %s from peer, error: %v", key, err)
			}
		}
		log.Printf("[GeeCache] Loading key: %s locally", key)
		// 没有找到合适的远程节点，或者从远程节点获取数据失败，则调用 g.getLocally(key) 进行本地获取。
		return g.getLocally(key)
	})
	if err == nil {
		return viewi.(data.ByteView), nil
	}
	return data.ByteView{}, fmt.Errorf("failed to load key: %s, error: %v", key, err)

}

// getLocally 调用用户回调函数 g.getter.Get() 获取源数据，
func (g *Group) getLocally(key string) (data.ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return data.ByteView{}, err
	}

	// 将源数据添加到缓存 mainCache 中（通过 populateCache 方法）
	value := data.ByteView{B: data.CloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, value data.ByteView) {
	g.maincache.Add(key, value)
}

// RegisterPeers registers a PeerPicker for choosing remote peer
// 为 Group 提供了一个选择远程缓存节点的机制，之后可以通过 PeerPicker 选择合适的节点来处理缓存请求。
func (g *Group) RegisterPeers(peers interfaces.PeerPicker) {
	// 对重复注册 PeerPicker 的检查，确保系统中每个 Group 只允许注册一次 peers。
	if g.peers != nil {
		panic("Register PeerPicker called more than once")
	}
	//允许 Group 通过 PeerPicker 来挑选远程节点处理缓存请求。
	g.peers = peers
}

func (g *Group) getFromPeer(peer interfaces.PeerGetter, key string) (data.ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	err := peer.Get(req, res)
	if err != nil {
		return data.ByteView{}, err
	}
	return data.ByteView{B: res.Value}, nil
}

// IncrementKeyUsage 统计 Key 的访问次数
func (g *Group) IncrementKeyUsage(key string) {
	g.hotKeyMutex.Lock()
	defer g.hotKeyMutex.Unlock()
	// 将 hotKeys 的初始化提前到 Group 的构造函数中。
	// if g.hotKeys == nil {
	// 	g.hotKeys = make(map[string]int)
	// }
	g.hotKeys[key]++
}

// IsHotKey 判断 Key 是否为热点
func (g *Group) IsHotKey(key string) bool {
	g.hotKeyMutex.RLock() // 使用读锁
	defer g.hotKeyMutex.RUnlock()
	count, exists := g.hotKeys[key]
	return exists && count > 100 // 假设访问次数超过 100 为热点
}

func (g *Group) SyncHotKeyToPeers(key string, value data.ByteView) error {
	// 如果当前 key 不是热点，直接返回
	if !g.IsHotKey(key) {
		return nil
	}

	/// 使用 ReplicatedPeerPicker 接口获取多个副本节点
	replicatedPicker, ok := g.peers.(interfaces.ReplicatedPeerPicker)
	if !ok {
		return fmt.Errorf("peers is not of type ReplicatedPeerPicker")
	}

	// 获取多个副本节点
	peers := replicatedPicker.GetReplicatedPeers(key, 3)
	if len(peers) == 0 {
		return fmt.Errorf("no peers available for key: %s", key)
	}
	var wg sync.WaitGroup
	var syncError error

	for _, peer := range peers {
		wg.Add(1)
		go func(p interfaces.PeerGetter) {
			defer wg.Done()
			req := &pb.Request{Group: g.name, Key: key}
			if err := p.Get(req, &pb.Response{}); err != nil {
				log.Printf("[GeeCache] Failed to sync key: %s to peer: %v, error: %v", key, p, err)
				syncError = err
			}
		}(peer)
	}
	wg.Wait()
	return syncError
}

func (g *Group) populateCacheOnPeer(peer interfaces.PeerGetter, key string, value data.ByteView) error {
	req := &pb.Request{Group: g.name, Key: key} // 修正没有 Value 字段的问题
	res := &pb.Response{}
	if err := peer.Get(req, res); err != nil {
		return err
	}

	// 将同步数据添加到远程节点的缓存
	remoteValue := data.ByteView{B: res.Value}
	g.maincache.Add(key, remoteValue)
	return nil
}
