package common

// Value 是一个用于缓存的通用接口，用于表示缓存中的值
type Value interface {
	Len() int
}
