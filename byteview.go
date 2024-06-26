package geecache

// A ByteView holds an immutable view of bytes.
// 只读数据结构 ByteView,表示缓存值
/*
使用 ByteView，你可以提供一个安全的、不可变的字节数据视图，
这在数据完整性和安全性至关重要的各种编程场景中特别有用。
*/
type ByteView struct {
	b []byte // b 将会存储真实的缓存值。byte 类型能够支持任意的数据类型的存储
}

func (v ByteView) Len() int {
	return len(v.b)
}

// 以字节切片的形式返回一个数据副本,确保 ByteView 中的原始数据不被更改。
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

// 有必要的话，制作一个字符串数据副本
func (v ByteView) String() string {
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
