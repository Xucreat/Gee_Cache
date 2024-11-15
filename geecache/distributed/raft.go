package distributed

/*实现 Raft 算法，节点的选举、日志复制、心跳等*/
import (
	"GeeCache/geecache/geecachepb"
	"context"
	"fmt"
	"hash/crc32"
	"sync"
	"time"

	"google.golang.org/grpc"
)

const (
	Follower = iota
	Candidate
	Leader
)

type Raft struct {
	mu          sync.Mutex
	state       int32
	currentTerm int32
	votedFor    int32
	log         []string
	commitIndex int32
	lastApplied int32
	peers       []string
	id          int32 // 在 Raft 算法中使用整数类型，保证节点的唯一性
	grpcServer  *grpc.Server
	grpcClients map[string]geecachepb.GroupCacheClient
}

func NewRaft(self string, peers []string) *Raft {
	// 将 self 转换为唯一的 int id (可以使用哈希函数，确保唯一性)
	id := int32(crc32.ChecksumIEEE([]byte(self))) // 通过 CRC32 哈希生成唯一 ID
	r := &Raft{
		id:          id,
		peers:       peers,
		state:       Follower,
		currentTerm: 0,
		votedFor:    -1,
		grpcClients: make(map[string]geecachepb.GroupCacheClient),
	}

	for _, peer := range peers {
		conn, err := grpc.Dial(peer, grpc.WithInsecure())
		if err != nil {
			fmt.Printf("Error connecting to peer %s: %v", peer, err)
			return nil
		}
		client := geecachepb.NewGroupCacheClient(conn)
		r.grpcClients[peer] = client
	}
	return r
}

// 启动 Raft 节点并开始选举
func (r *Raft) Start() {
	go r.runElection()
}

// runElection 是选举过程的实现
func (r *Raft) runElection() {
	for {
		time.Sleep(time.Second * 5)

		r.mu.Lock()
		if r.state == Leader {
			r.mu.Unlock()
			continue
		}

		// 尝试发起选举
		r.state = Candidate
		r.currentTerm++
		r.votedFor = r.id
		r.mu.Unlock()

		votes := 1 // 当前节点投票给自己
		for _, peer := range r.peers {
			if peer == fmt.Sprintf("localhost:%d", r.id) {
				continue
			}

			go r.sendVoteRequest(peer, &votes)
		}

		// 等待选举完成的判断
		time.Sleep(time.Second * 2)
		if votes > len(r.peers)/2 {
			r.mu.Lock()
			r.state = Leader
			r.mu.Unlock()
			fmt.Printf("Node %d became leader\n", r.id)
			r.sendHeartbeats()
			break
		}
	}
}

// sendVoteRequest 发送投票请求
func (r *Raft) sendVoteRequest(peer string, votes *int) {
	conn, err := grpc.Dial(peer, grpc.WithInsecure())
	if err != nil {
		fmt.Println("Failed to connect to peer:", peer)
		return
	}
	defer conn.Close()

	client := geecachepb.NewGroupCacheClient(conn)
	req := &geecachepb.RequestVoteRequest{
		Term:        r.currentTerm,
		CandidateId: r.id,
	}

	res, err := client.RequestVote(context.Background(), req)
	if err != nil {
		fmt.Println("Failed to send vote request:", err)
		return
	}

	if res.VoteGranted {
		fmt.Printf("Vote granted by %s\n", peer)
		(*votes)++
	}
}

// sendHeartbeats 领导者定期发送心跳
func (r *Raft) sendHeartbeats() {
	for {
		if r.state != Leader {
			return
		}

		for _, peer := range r.peers {
			if peer == fmt.Sprintf("localhost:%d", r.id) {
				continue
			}
			go r.sendHeartbeat(peer)
		}

		time.Sleep(time.Second * 1)
	}
}

// sendHeartbeat 发送心跳
func (r *Raft) sendHeartbeat(peer string) {
	conn, err := grpc.Dial(peer, grpc.WithInsecure())
	if err != nil {
		fmt.Println("Failed to connect to peer:", peer)
		return
	}
	defer conn.Close()

	client := geecachepb.NewGroupCacheClient(conn)
	req := &geecachepb.AppendEntriesRequest{
		Term:     r.currentTerm,
		LeaderId: r.id,
	}

	_, err = client.AppendEntries(context.Background(), req)
	if err != nil {
		fmt.Println("Failed to send heartbeat:", err)
	}
}
