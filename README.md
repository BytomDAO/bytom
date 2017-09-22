# Bytom(support utxo,bvm,p2p,pow,account,asset,grpc,json http.)

## clone code
``` bash
git clone https://github.com/Bytom/bytom $GOPATH/src/github.com
```
## build bytom
``` bash
1. make install
2. cd ./cmd/bytom
3. go build
```
## build bytomcli
``` bash
1. cd ./cmd/bytomcli
2. go build
```
## p2p & grpc test (两个节点测试)
``` bash
1. cd ./cmd/bytom
2. ./test.sh bytom0
3. ./test.sh bytom1
4. curl -X POST --data '{"jsonrpc":"2.0", "method": "net_info", "params":[], "id":"67"}' http://127.0.0.1:46657
```
## bytomcli & bytom test
``` bash
1. cd ./cmd/bytom
2. ./test.sh bytom0
3. cd ./cmd/bytomcli
4. ./bytomcli <command> <opt...>
```
