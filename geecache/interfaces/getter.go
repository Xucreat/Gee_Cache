package interfaces

// A Getter loads data for a key
type Getter interface {
	Get(key string) ([]byte, error)
}

// A Getter implements Getter with a function
type GetterFunc func(key string) ([]byte, error)

// Get implements Getter interface function
// 函数类型实现某一个接口，称之为接口型函数，方便使用者在调用时既能够传入函数作为参数，
// 也能够传入实现了该接口的结构体作为参数。
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}
