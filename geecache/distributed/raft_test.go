package distributed_test

import (
	"context"
	"fmt"
	"log"
	"net"
	"testing"
	"time"

	"GeeCache/geecache/geecachepb"

	"google.golang.org/grpc"
)

type mockRaftServer struct {
	geecachepb.UnimplementedGroupCacheServer
}

func (s *mockRaftServer) RequestVote(ctx context.Context, req *geecachepb.RequestVoteRequest) (*geecachepb.RequestVoteResponse, error) {
	log.Printf("Received vote request from candidate %d for term %d", req.CandidateId, req.Term)
	return &geecachepb.RequestVoteResponse{
		VoteGranted: true,
	}, nil
}

func (s *mockRaftServer) AppendEntries(ctx context.Context, req *geecachepb.AppendEntriesRequest) (*geecachepb.AppendEntriesResponse, error) {
	log.Printf("Received heartbeat from leader %d for term %d", req.LeaderId, req.Term)
	return &geecachepb.AppendEntriesResponse{
		Success: true,
	}, nil
}

func startGRPCServer(t *testing.T) string {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	geecachepb.RegisterGroupCacheServer(grpcServer, &mockRaftServer{})

	// 创建一个通道用于传递错误
	errChan := make(chan error, 1)

	// 启动 Goroutine 执行 gRPC 服务器
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			errChan <- fmt.Errorf("failed to serve: %v", err)
		}
	}()

	// 等待 gRPC 服务器启动并能接受连接:
	// 放在线程中，阻塞err = <-errChan。因为当启动成功后，errchan不会有数据，
	// 就会阻塞在err = <-errChan这一步，直到另一个线程完成延时后，往里面输入nil。
	go func() {
		time.Sleep(500 * time.Millisecond) // 等待 500 毫秒，确保服务器已启动
		errChan <- nil
	}()

	// 在主测试 Goroutine 中接收错误并处理
	err = <-errChan
	if err != nil {
		t.Fatalf("gRPC server failed: %v", err) // 确保 t.Fatalf 在主 Goroutine 中调用
	}
	return ":50051"
}

// 模拟 Raft 节点
type Raft struct {
	id          int32
	currentTerm int32
	peers       []string
}

func (r *Raft) sendVoteRequest(peer string) {
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
	}
}

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

func TestRaft(t *testing.T) {
	// 启动 gRPC 服务端
	serverAddr := startGRPCServer(t)

	// 创建 Raft 节点
	raftNode := &Raft{
		id:          1,
		currentTerm: 1,
		peers:       []string{serverAddr},
	}

	// 模拟发送投票请求
	raftNode.sendVoteRequest(serverAddr)

	// 模拟发送心跳
	raftNode.sendHeartbeat(serverAddr)
}
