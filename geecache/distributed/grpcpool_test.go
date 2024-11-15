// // grpcpool_test.go
package distributed_test

// import (
// 	"GeeCache/geecache/geecachepb"
// 	"context"
// 	"fmt"
// 	"log"
// 	"net"
// 	"testing"
// 	"time"

// 	"google.golang.org/grpc"
// )

// // 定义一个测试服务器实现
// type testServer struct {
// 	geecachepb.UnimplementedGroupCacheServer
// 	data map[string]string
// }

// func (s *testServer) Get(ctx context.Context, req *geecachepb.Request) (*geecachepb.Response, error) {
// 	log.Printf("Received gRPC Get request for key: %s", req.GetKey()) // 添加日志
// 	value, ok := s.data[req.Key]
// 	if !ok {
// 		return nil, fmt.Errorf("key not found: %s", req.Key)
// 	}
// 	log.Printf("Returning value: %s for key: %s", value, req.GetKey()) // 添加日志
// 	return &geecachepb.Response{Value: []byte(value)}, nil
// }

// func startTestGRPCServer(addr string, data map[string]string) (*grpc.Server, error) {
// 	lis, err := net.Listen("tcp", addr)
// 	if err != nil {
// 		return nil, err
// 	}
// 	grpcServer := grpc.NewServer()
// 	geecachepb.RegisterGroupCacheServer(grpcServer, &testServer{data: data})
// 	go func() {
// 		if err := grpcServer.Serve(lis); err != nil {
// 			log.Fatalf("failed to serve: %v", err)
// 		}
// 	}()
// 	return grpcServer, nil
// }

// func TestGRPCPool_SetAndPickPeer(t *testing.T) {
// 	data := map[string]string{
// 		"Tom":  "630",
// 		"Jack": "589",
// 		"Sam":  "567",
// 	}

// 	// 启动测试 gRPC 服务
// 	addr1 := "localhost:50051"
// 	server1, err := startTestGRPCServer(addr1, data)
// 	if err != nil {
// 		t.Fatalf("Failed to start gRPC server: %v", err)
// 	}
// 	defer server1.Stop()

// 	// 等待服务器启动
// 	time.Sleep(500 * time.Millisecond)

// 	// 初始化 GRPCPool，设置一个不同的 self 地址
// 	selfAddr := "localhost:50052" // 确保这个地址不同于 addr1
// 	pool := NewGRPCPool(selfAddr)
// 	pool.Set(addr1)

// 	// 测试 PickPeer 功能
// 	peer, ok := pool.PickPeer("Tom")
// 	if !ok || peer == nil {
// 		t.Fatalf("Failed to pick peer for key 'Tom'")
// 	}

// 	// 测试 grpcGetter 的 Get 功能
// 	request := &geecachepb.Request{
// 		Group: "testGroup",
// 		Key:   "Tom",
// 	}
// 	response := &geecachepb.Response{}
// 	err = peer.Get(request, response)
// 	if err != nil {
// 		t.Fatalf("Failed to get value from peer: %v", err)
// 	}

// 	if string(response.Value) != data["Tom"] {
// 		t.Errorf("Expected %s, got %s", data["Tom"], string(response.Value))
// 	}
// }

// func TestGRPCClient_Get(t *testing.T) {
// 	data := map[string]string{
// 		"Alice": "100",
// 		"Bob":   "200",
// 	}

// 	// 启动测试 gRPC 服务
// 	addr := "localhost:50052"
// 	server, err := startTestGRPCServer(addr, data)
// 	if err != nil {
// 		t.Fatalf("Failed to start gRPC server: %v", err)
// 	}
// 	defer server.Stop()

// 	// 等待服务器启动
// 	time.Sleep(500 * time.Millisecond)

// 	// 创建 gRPC 客户端
// 	client, err := NewGRPCClient(addr)
// 	if err != nil {
// 		t.Fatalf("Failed to create gRPC client: %v", err)
// 	}

// 	// 测试 Get 方法
// 	value, err := client.Get("testGroup", "Alice")
// 	if err != nil {
// 		t.Fatalf("Failed to get value: %v", err)
// 	}
// 	if string(value) != data["Alice"] {
// 		t.Errorf("Expected %s, got %s", data["Alice"], string(value))
// 	}
// }
