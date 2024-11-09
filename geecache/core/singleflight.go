/*使用 singleflight 防止缓存击穿*/
package core

import "sync"

// call 代表正在进行中，或已经结束的请求
type call struct {
	wg  sync.WaitGroup // 使用 sync.WaitGroup 锁避免重入
	val interface{}
	err error
}

// 管理不同 key 的请求(call)
type RequestGroup struct {
	mu sync.Mutex       // 保护m
	m  map[string]*call // 管理不同 key 的请求
}

func (g *RequestGroup) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
