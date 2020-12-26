# 信通院安全评测版本

```shell script
root
123456

make bytomd
make install

bytomd init --chain_id testnet
bytomd node --auth.disable

curl -k https://localhost:9888/net-info
localhost 会解析为 ip6 ::1
curl -k https://127.0.0.1:9888/net-info
127.0.0.1 会解析为 ip4 127.0.0.1
```

```toml
[api]
enable_tls = true
cert_file = "key/cert.pem"
key_file = "key/key.pem"
white_list = ["127.0.0.1"]
black_list = ["::1"]
```
