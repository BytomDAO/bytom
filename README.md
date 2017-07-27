# BlockChain.

在终端键入
1. make install
2. cd ./cmd/blockchain
3. go build
编译成功

两个节点测试
在终端键入
1. ./test.sh node1
2. ./test.sh node2


rpc test:

curl -X POST --data '{"jsonrpc":"2.0", "method": "net_info", "params":[], "id":"67"}' http://127.0.0.1:46657
