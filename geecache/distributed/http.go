package distributed

import (
	"GeeCache/geecache/core"
	pb "GeeCache/geecache/geecachepb"
	"GeeCache/geecache/interfaces"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"google.golang.org/protobuf/proto"
)

// const defaultBasePath = "/_GeeCache/geecache/"

// // HTTPPool 结构体是用于管理缓存节点之间 HTTP 通信的池;
// // HTTPPool 实现了服务端功能
// // self 表示当前节点的基础 URL。例如 https://example.net:8000，这是该节点对外暴露的地址。
// // basePath 用于表示处理缓存请求的 HTTP 路径前缀，默认值为 /_GeeCache/geecache/。所有与缓存相关的 HTTP 请求都应该以这个前缀开头。
// type HTTPPool struct {
// 	self     string
// 	basePath string
// }

/*为 HTTPPool 添加节点选择的功能*/
const (
	defaultBasePath = "/_geeccache/"
	defaultReplicas = 50
)

// HTTPPool implements PeerPicker for a pool of HTTP peers
type HTTPPool struct {
	// this peer's base URL, e.g. "https://example.net:8000"
	self     string
	basePath string
	mu       sync.Mutex // guards peer and httpGetters
	peers    *Map       // 类型是一致性哈希算法的 Map，用来根据具体的 key 选择节点。

	// 映射远程节点与对应的 httpGetter。
	// 每一个远程节点对应一个 httpGetter，因为 httpGetter 与远程节点的地址 baseURL 有关。
	httpGetters map[string]*httpGetter // keyed by e.g. "http://10.0.0.2:8008"
}

// NewHTTPPool initializes an HTTP pool of peers.
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// log info with server name
func (p *HTTPPool) log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP handle all http requests
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 判断访问路径的前缀是否是 basePath，不是返回错误。
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.log("%s %s", r.Method, r.URL.Path)
	// <basepath>/<groupname>/<key> required

	// SplitN 将 s 切片为由 sep 分隔的子字符串，并返回这些分隔符之间的子字符串切片。
	// 计数确定要返回的子字符串的数量：
	// n > 0：至多n个子串；最后一个子字符串将是未分割的余数。
	// n == 0：结果为零（零个子串）
	// n < 0：所有子串
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	// 通过 groupname 得到 group 实例
	group := core.GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	// 使用 group.Get(key) 获取缓存数据。
	view, _ := group.Get(key)
	// 使用 proto.Marshal() 编码 HTTP 响应
	body, err := proto.Marshal(&pb.Response{Value: view.ByteSlice()})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	// 使用 w.Write() 将缓存值以字节切片的形式,作为 httpResponse 的 body 返回。
	w.Write(body)
}

/*创建具体的 HTTP 客户端类 httpGetter，实现 PeerGetter 接口*/
// 结构体的主要目的是用来从远程节点获取缓存数据。
type httpGetter struct {
	baseURL string // 远程节点的基本 URL 地址
}

// Get 方法负责根据 group 和 key 构造请求，并通过 HTTP 请求获取数据，处理错误并返回结果。
func (h *httpGetter) Get(in *pb.Request, out *pb.Response) error {
	// 构造一个请求的 URL
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		// 处理字符串，以安全地放入URL query中
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()),
	)
	// Get 向指定的 URL 发出 GET。
	// 如果响应是以下重定向代码之一，则 Get 遵循重定向，最多 10 个重定向：
	// 调用 http.Get(u) 发起对远程服务器的 HTTP 请求，获取与 group 和 key 相关的数据。
	res, err := http.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// 发起 HTTP 请求出错或服务器返回状态码不是 200 OK，就会返回错误信息。
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", res.Status)
	}

	// 读取响应体的内容，将它存储为字节数组 bytes
	bytes, _ := io.ReadAll(res.Body)
	// 使用 proto.Unmarshal() 解码 HTTP 响应
	if err = proto.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding response body: %v", err)
	}

	return nil
}

// 检查 httpGetter 是否实现了 PeerGetter 接口
var _ interfaces.PeerGetter = (*httpGetter)(nil)

/*实现 PeerPicker 接口*/
// Set updates the pool's list of peers
// 通过 Set 方法，HTTPPool 能够动态更新集群中的 peers 列表，确保分布式缓存系统的扩展性。
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// p.peers 被初始化为一个一致性哈希的实例
	p.peers = New(defaultReplicas, nil)
	// 将传入的 peers 添加到一致性哈希环中
	p.peers.Add(peers...)
	// p.httpGetters 是一个 map，用于存储每个 peer 对应的 httpGetter 实例
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	// 个 peer 对应一个 httpGetter，用于与远程节点通信
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath} // peer 的基础地址加上 basePath，构成一个完整的请求 URL。
	}
}

// PickPeer picks a peer according to key
// 根据传入的 key 来选择一个合适的 peer（节点），并返回该节点对应的 httpGetter（用于与该节点通信）。
func (p *HTTPPool) PickPeer(key string) (interfaces.PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// p.peers.Get(key),根据传入的key，通过一致性哈希算法返回一个真实的节点; 确保找到的 peer 不是当前节点
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

var _ interfaces.PeerPicker = (*HTTPPool)(nil)
