module GeeCachecd

go 1.23.0

replace geecache => ./geecache // 指向子目录的 geecache 包

require geecache v0.0.0 // 依赖声明，指向本地 geecache 目录

require google.golang.org/protobuf v1.34.2 // indirect
