package interfaces

import (
	"GeeCache/geecache/common"
)

// // Value represents a value stored in the cache
// type Value interface {
// 	Len() int
// }

// 接口定义
type EvictionPolicy interface {
	Get(key string) (value common.Value, ok bool)
	Add(key string, value common.Value)
	RemoveOldest()
	Len() int
}
