package interfaces

import "GeeCache/geecache/data"

// Value represents a value stored in the cache
type Value interface {
	Len() int
}

// 接口定义
type EvictionPolicy interface {
	Get(key string) (value data.ByteView, ok bool)
	Add(key string, value data.ByteView)
	RemoveOldest()
	Len() int
}
