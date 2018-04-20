
# 发送远端交易
## Example
```
$ go build
$ export BYTOM_URL="http://192.168.199.62:9888"
$ ./sendbulktx

tx num: 600, use time: 3m8.8846942s
height: 561 ,tx num: 67
height: 562 ,tx num: 67
height: 563 ,tx num: 51
height: 564 ,tx num: 156
height: 565 ,tx num: 82
height: 559 ,tx num: 17
height: 560 ,tx num: 160

```
available flags for `sendbulktx`:

```
      --assetnum int    Number of transactions asset (default 10)
      --config string   config file (default "./config.toml")
      --thdnum int      goroutine num (default 5)
      --thdtxnum int     The number of transactions per goroutine (default 10)
```

# config.toml
```
send_acct_id = "0CMUIQ06G0A02"
send_asset_id = "36017df0de65f4de249c966b9a98b8765ee9ecd438be14cdefce9b6467e7a752"
#"issue", "spend", "address"
build_type = "address"
#asset_receiver = ["bm1qm97wwnjvgxarwzgd4q9saf38fj9r5jr8aelcvp"]
asset_receiver = ["bm1q678m0eac5xcxxvalynzjp5cl4wq06rp039svzs","bm1qhgpjl0f7hzyxj866v0ztllscw6uqmh5gfcuqgy"]
```
