package tests

import (
	"GeeCache/geecache/core"
	"GeeCache/geecache/interfaces"
	"fmt"
	"log"
	"reflect"
	"testing"
)

type Getter interfaces.Getter

/*借助 GetterFunc 的类型转换，将一个匿名回调函数转换成了接口 f Getter
调用该接口的方法 f.Get(key string)，实际上就是在调用匿名回调函数。*/

func TestGetter(t *testing.T) {
	var f Getter = interfaces.GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})

	expect := []byte("key")
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, expect) {
		t.Errorf("callback faild")
	}
}

/*
测试用例,如何使用实现的单机并发缓存。

该测试用例首先检查缓存为空时是否能正确获取原始数据，
然后验证在缓存存在的情况下，
是否能够避免再次调用原始数据源，确保缓存系统的正常工作。
*/
var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func TestGet(t *testing.T) {
	loadCounts := make(map[string]int, len(db))
	gee := core.NewGroup("scores", 2<<10, interfaces.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				if _, ok := loadCounts[key]; !ok {
					loadCounts[key] = 0
				}
				loadCounts[key] += 1
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	for k, v := range db {
		// 调用 gee.Get(k) 从 缓存 获取该键的值
		// 在缓存为空的情况下，能够通过回调函数获取到源数据并放入缓存。
		if view, err := gee.Get(k); err != nil || view.String() != v {
			t.Fatal("failed to get value of Tom")
		} // load from callback function

		// 在缓存已经存在的情况下，是否直接从缓存中获取(上条if语句已经完成了:若在缓存中没找到则调用回调函数的功能)
		/*为了实现这一点，使用 loadCounts 统计某个键调用回调函数的次数，
		如果次数大于1，则表示调用了多次回调函数，说明缓存未命中。*/
		if _, err := gee.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		} // cache hit
	}

	if view, err := gee.Get("unknow"); err == nil {
		t.Fatalf("the value of unknow should be empty, but %s got", view)
	}

}
