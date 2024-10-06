#!/bin/bash
trap "rm server;kill 0" EXIT # trap 命令用于在 shell 脚本退出时，删掉临时文件，结束子进程。

# 编译当前目录下的 Go 代码，并将输出的可执行文件命名为 server
go build -o server
# 启动多个 Geecache 实例;通过 & 符号让每个实例在后台运行
./server -port=8001 &
./server -port=8002 &
# 启动第三个实例，监听 8003 端口，并启用 API 服务器（通过 -api=1 参数）
./server -port=8003 -api=1 &

sleep 2
# 向终端输出一条提示信息:通知用户，缓存系统已经启动完毕，接下来会开始进行测试
# 如果在脚本中有多个步骤，通过 echo 命令可以知道执行的进度和脚本卡在哪个步骤，有助于调试。
echo ">>> start test"
# 测试缓存系统：测试 API 服务器的响应：并发发送 3 个请求，查询缓存中的 Tom 键。
curl "http://localhost:9999/api?key=Tom" &
curl "http://localhost:9999/api?key=Tom" &
curl "http://localhost:9999/api?key=Tom" &

wait