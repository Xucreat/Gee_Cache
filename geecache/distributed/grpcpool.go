package distributed

import (
	"GeeCache/geecache/core"
	"GeeCache/geecache/geecachepb"
	"GeeCache/geecache/interfaces"
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

/*为 HTTPPool 添加节点选择的功能*/
const (
	defaultBasePath = "/_geeccache/"
	defaultReplicas = 50
)

// GRPCPool 用于管理 gRPC 节点池，并提供通信接口
type GRPCPool struct {
	self        string                 //  保持为字符串类型，可以代表节点的主机地址、域名或其他唯一标识符
	mu          sync.Mutex             // guards peer and grpcGetters
	peers       *Map                   // 一致性哈希映射，选择节点
	grpcClients map[string]*grpcClient // 每个节点对应的 gRPC 客户端
}

// NewGRPCPool 初始化一个 gRPC 节点池
func NewGRPCPool(self string) *GRPCPool {
	return &GRPCPool{
		self: self,
	}
}

// Set 更新节点池中的节点并启动 Raft 算法
func (p *GRPCPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 一致性哈希映射
	p.peers = New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.grpcClients = make(map[string]*grpcClient, len(peers))

	// 启动 Raft 算法
	raft := NewRaft(p.self, peers)
	go raft.Start()

	// 为每个 peer 创建 gRPC 连接
	for _, peer := range peers {
		conn, err := grpc.Dial(
			peer,
			grpc.WithTransportCredentials(insecure.NewCredentials()), // 使用不加密的连接
		)
		if err != nil {
			log.Fatalf("failed to connect to peer %s: %v", peer, err)
		}
		client := geecachepb.NewGroupCacheClient(conn)
		p.grpcClients[peer] = &grpcClient{client: client} // 将 gRPC 客户端封装成 grpcClient
	}
}

// PickPeer 根据 key 选择对应的 peer
func (p *GRPCPool) PickPeer(key string) (interfaces.PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		log.Printf("Pick peer %s", peer)
		return p.grpcClients[peer], true // 返回 grpcGetter
	}
	return nil, false
}

// grpcClient 用于从远程节点获取缓存数据
type grpcClient struct {
	client geecachepb.GroupCacheClient // gRPC 客户端
}

// Get 实现 PeerGetter 接口，用于通过 gRPC 获取缓存数据
func (g *grpcClient) Get(in *geecachepb.Request, out *geecachepb.Response) error {
	// 使用 g.client 发送 gRPC 请求
	res, err := g.client.Get(context.Background(), in)
	if err != nil {
		return fmt.Errorf("failed to get: %v", err)
	}
	out.Value = res.Value // 将返回的数据赋值给 out: 避免复制包含 sync.Mutex 的结构体，尤其是在并发环境下，应该始终通过指针传递这些结构体。
	return nil
}

// 检查 grpcClient 是否实现了 PeerGetter 接口
var _ interfaces.PeerGetter = (*grpcClient)(nil)

// 实现 gRPC 服务器
type server struct {
	geecachepb.UnimplementedGroupCacheServer
}

func (s *server) Get(ctx context.Context, req *geecachepb.Request) (*geecachepb.Response, error) {
	groupName := req.GetGroup()
	key := req.GetKey()

	group := core.GetGroup(groupName)
	if group == nil {
		return nil, fmt.Errorf("group not found: %s", groupName)
	}

	// 获取缓存数据
	view, err := group.Get(key)
	if err != nil {
		return nil, fmt.Errorf("error getting key: %v", err)
	}

	return &geecachepb.Response{Value: view.ByteSlice()}, nil
}

// 启动 gRPC 服务器
func StartGRPCServer(addr string) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()

	// 注册 GroupCache 服务
	geecachepb.RegisterGroupCacheServer(grpcServer, &server{})

	log.Printf("gRPC server listening at %v", addr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// NewGRPCClient 创建 gRPC 客户端并与远程服务器建立连接
func NewGRPCClient(addr string) (*grpcClient, error) {
	// 使用 grpc.DialContext，并替代 WithInsecure
	conn, err := grpc.DialContext(
		context.Background(),
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()), // 使用不加密的连接
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %v", err)
	}

	client := geecachepb.NewGroupCacheClient(conn)
	return &grpcClient{client: client}, nil
}

// Get 获取缓存数据
func (c *grpcClient) GetData(group, key string) ([]byte, error) {
	req := &geecachepb.Request{
		Group: group,
		Key:   key,
	}
	resp, err := c.client.Get(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed to get: %v", err)
	}
	return resp.GetValue(), nil
}

func (p *GRPCPool) GetReplicatedPeers(key string, replicas int) []interfaces.PeerGetter {
	p.mu.Lock()
	defer p.mu.Unlock()

	peerNames := p.peers.GetMultipleNodes(key, replicas)
	var getters []interfaces.PeerGetter
	for _, peer := range peerNames {
		if client, exists := p.grpcClients[peer]; exists {
			getters = append(getters, client)
		}
	}
	return getters
}

// 确保 GRPCPool 实现了 ReplicatedPeerPicker 接口
var _ interfaces.ReplicatedPeerPicker = (*GRPCPool)(nil)

// SelectPeer 从多个副本中选择一个节点
func (p *GRPCPool) SelectPeer(peers []interfaces.PeerGetter) interfaces.PeerGetter {
	if len(peers) == 0 {
		return nil
	}
	return peers[rand.Intn(len(peers))] // 随机选择一个副本
}
