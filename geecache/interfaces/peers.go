/*1.注册节点(Register Peers)，借助一致性哈希算法选择节点*/
/*2.实现 HTTP 客户端，与远程节点的服务端通信*/
package interfaces

import (
	pb "GeeCache/geecache/geecachepb" // 新的 geecachepb 包路径
)

// PeerPicker is the interface that must be implement to locate
// the peer that owns a specific key
// 根据传入的 key 选择相应节点
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

// PeerGetter is the interface that must be implemented by a peer
// 从对应 group 查找缓存值。
// PeerGetter 就对应于上述流程中的 HTTP 客户端。
type PeerGetter interface {
	Get(in *pb.Request, out *pb.Response) error
}
