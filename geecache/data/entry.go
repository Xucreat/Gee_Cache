package data

import (
	"GeeCache/geecache/common"
)

// entry represents a key-value pair along with its frequency or other metadata.
type Entry struct {
	Key       string       // key of the entry
	Value     common.Value // value of the entry
	Frequency int          // access frequency of the entry (for LFU)
}
