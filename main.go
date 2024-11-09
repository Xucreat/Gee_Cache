/* day3-http-server/main.go:实现 main 函数，实例化 group，并启动 HTTP 服务。*/
// package main

// import (
// 	"fmt"
// 	"geecache"
// 	"log"
// 	"net/http"
// )

// var db = map[string]string{
// 	"Tom":  "630",
// 	"Jack": "589",
// 	"Sam":  "567",
// }

// func main() {
// 	// 创建一个名为 scores 的 Group，若缓存为空，回调函数会从 db 中获取数据并返回。
// 	geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(
// 		func(key string) ([]byte, error) {
// 			log.Println("[SlowDB] search key", key)
// 			if v, ok := db[key]; ok {
// 				return []byte(v), nil
// 			}
// 			// 当key既不存在与缓存也不存在于db中时输出日志信息到控制台
// 			log.Println("[SlowDB] search no key", key)
// 			return nil, fmt.Errorf("%s not exist", key)
// 		}))

// 	addr := "localhost:9999"
// 	peers := geecache.NewHTTPPool(addr)
// 	log.Println("geecache is running at", addr)
// 	log.Fatal(http.ListenAndServe(addr, peers))
// }

/*day5-multi-nodes/main.go*/
package main

import (
	core "GeeCache/geecache/core"
	"GeeCache/geecache/distributed" // 引入分布式功能
	interfaces "GeeCache/geecache/interfaces"
	"flag"
	"fmt"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func createGroup() *core.Group {
	return core.NewGroup("scores", 2<<10, interfaces.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not eixst", key)
		}))
}

func startCacheServer(addr string, addrs []string, gee *core.Group) {
	peers := distributed.NewHTTPPool(addr)
	peers.Set(addrs...)
	gee.RegisterPeers(peers)
	log.Println("geecache is running  at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

func startAPIServer(apiAddr string, gee *core.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := gee.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())

		}))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

func main() {
	var port int
	var api bool
	// 通过命令行参数 -port 设置缓存服务器的端口，默认端口是 8001。
	flag.IntVar(&port, "port", 8001, "Geecache server port")
	// 通过 -api 参数决定是否启动 API 服务器，默认不启动。
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	// 定义 API 服务器地址：
	// API 服务器的地址是固定的 http://localhost:9999，用于接收外部请求或提供管理接口。
	apiAddr := "http://localhost:9999"
	// 缓存服务器地址映射：
	// addrMap 定义了 3 个缓存服务器的地址，端口分别为 8001、8002、8003，用于支持分布式缓存系统中的多个节点。
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	// 通过遍历 addrMap，将所有服务器的地址存入 addrs 数组，
	// 方便后续在 startCacheServer 中用于节点之间的通信。
	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	// 创建缓存组：
	// 创建一个缓存组实例 gee，负责管理缓存数据。
	gee := createGroup()
	// 启动 API 服务器:与用户进行交互，用户感知
	// 如果命令行参数中指定了 -api，则启动 API 服务器。
	if api {
		// 使用 go 关键字开启一个协程，后台运行 startAPIServer(apiAddr, gee)，不阻塞主进程。
		go startAPIServer(apiAddr, gee)
	}
	// 启动缓存服务器：
	// 并指定当前节点的地址（通过 addrMap[port] 获取）
	// 以及其他所有节点的地址（addrs 列表）
	// gee 表示缓存组实例，处理缓存数据的获取和存储。
	startCacheServer(addrMap[port], []string(addrs), gee)
}
