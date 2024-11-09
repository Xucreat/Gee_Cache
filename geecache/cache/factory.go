package cache

func NewCache(algorithm string, cacheBytes int64) Cache {
	switch algorithm {
	case "lru":
		return NewLRUCache(cacheBytes, nil) // 你需要实现此函数
	case "lfu":
		return NewLFUCache(int(cacheBytes)) // 你需要实现此函数
	default:
		return nil
	}
}
