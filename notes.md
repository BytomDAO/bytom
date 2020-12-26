# 信通院安全评测版本

```shell script
root
123456

make bytomd
make install

bytomd init --chain_id testnet
bytomd node --auth.disable

curl -k https://localhost:9888/net-info
```
