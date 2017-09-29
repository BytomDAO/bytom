Bytom
=====

[![AGPL v3](https://img.shields.io/badge/license-AGPL%20v3-brightgreen.svg)](./LICENSE)

## Table of Contents
<!-- vim-markdown-toc GFM -->

* [What is Bytom?](#what-is-bytom)
* [Build from source](#build-from-source)
  * [Requirements](#requirements)
  * [Installation](#installation)
    * [Get the source code](#get-the-source-code)
    * [Build](#build)
* [Example](#example)
  * [Set up a wallet and manage the key](#set-up-a-wallet-and-manage-the-key)
  * [Create and launch a single node](#create-and-launch-a-single-node)
    * [Asset issuance test](#asset-issuance-test)
  * [Multiple node](#multiple-node)
* [Contributing](#contributing)
* [License](#license)

<!-- vim-markdown-toc -->

## What is Bytom?

Bytom is software designed to operate and connect to highly scalable blockchain networks confirming to the Bytom Blockchain Protocol, which allows partipicants to define, issue and transfer digitial assets on a multi-asset shared ledger. Please refer to the [White Paper](https://github.com/Bytom/wiki/blob/master/White-Paper/%E6%AF%94%E5%8E%9F%E9%93%BE%E6%8A%80%E6%9C%AF%E7%99%BD%E7%9A%AE%E4%B9%A6-%E8%8B%B1%E6%96%87%E7%89%88.md) for more details.

In the current state `bytom` is able to:

- Issue assets
- Manage account as well as asset

## Build from source

### Requirements

- [Go](https://golang.org/doc/install) version 1.8 or higher, with `$GOPATH` set to your preferred directory

### Installation

Ensure Go with the supported version is installed properly:

```bash
$ go version
$ go env GOROOT GOPATH
```

#### Get the source code

``` bash
$ git clone https://github.com/Bytom/bytom $GOPATH/src/github.com/bytom
```

#### Build

- Bytom

``` bash
$ cd $GOPATH/src/github.com/bytom
$ make install
$ cd ./cmd/bytom
$ go build
```

- Bytomcli

```go
$ cd $GOPATH/src/github.com/bytom/cmd/bytomcli
$ go build
```

## Example

Currently, bytom is still in active development and a ton of work needs to be done, but we also provide the following content for these eager to do something with `bytom`.

### Set up a wallet and manage the key

You can create an account via `create-key password`, which will generate a `keystore` directory containing the keys under the project directory.

```bash
$ ./bytomcli create-key account_name password   # Create an account named account_name using password
$ ./bytomcli delete-key password pubkey         # Delete account pubkey
$ ./bytomcli reset-password oldpassword newpassword pubkey  # Update password
```

### Create and launch a single node

When successfully building the project, the `bytom` and `bytomcli` binary should be present in `cmd/bytom/bytom` and `cmd/bytomcli/bytomcli`, respectively. The next step is to initialize the node:

```bash
$ cd ./cmd/bytom
$ ./bytom init --home ./.bytom
```

After that, you'll see `.bytom` generated in current directory, then launch the single node:

``` bash
$ ./bytom node --home ./.bytom
```

#### Asset issuance test

Given the `bytom` node is running, the general workflow is as follows:

- create an account
- create an asset
- create/sign/submit a transaction to transfer an asset
- query the assets on-chain

Create an account named `alice`:

```bash
$ ./bytomcli create-account alice
xprv:<alice_account_private_key>
responses:{acc04K9MCFBG0A04 alice [0xc4200966e0] 1 0xc4204be220}
account id:<alice_account_id>
```

Create an asset named `gold`:

```bash
$ ./bytomcli create-asset gold
xprv:<gold_asset_private_key>
xpub:[98 55 100 48 102 100 53 101 54 55 55 49 48 52 97 100 50 100 51 51 98 49 56 98 98 100 55 55 50 51 98 53 102 51 101 97 56 55 49 52 48 53 57 54 50 56 55 48 49 97 50 99 97 100 101 51 51 102 100 100 97 53 56 51 49 54 97 50 57 54 101 49 102 100 48 102 53 57 99 55 50 49 53 98 50 54 55 102 50 52 102 52 54 50 48 101 51 48 102 55 99 51 50 56 49 102 97 52 99 55 97 53 102 50 57 97 100 53 51 100 56 100 55 56 50 50 98 98]
responses:[{{4131000809721133708 15036469059929217352 9712753415038655527 16992088508821480533} gold [118 107 170 32 152 106 231 249 212 15 215 121 94 191 102 23 231 61 38 211 121 176 221 199 48 173 145 207 243 201 82 0 215 2 72 243 81 81 173 105 108 0 192] [0xc420020850] 1 0xc4204c1960 0xc4204c1980 true}]
asset id:<gold_asset_id>
```

Now we can transafer 10000 gold to `alice` using a single command `sub-create-issue-tx`:

```bash
$ ./bytomcli sub-create-issue-tx <alice_account_id> <gold_asset_id> <asset_private_key> <gold_asset_amount>
To build transaction:
-----------tpl:{version:1 serialized_size:314 result_ids:<71c3b949750c887e466422007cdd1a6a9f3449e3bacd43307e361e84d76fe37b> data:<130994550772:/* unknown wire type 7 */ 1642:/* unknown wire type 7 */ 10:17681930801800169409 159728:7652 9:4897805654558278394 9:/* unexpected EOF */ >min_time_ms:1506587706078 max_time_ms:1506588006078  [0xc4204c9060 0xc4204c91e0] true false}
----------tpl transaction:version:1 serialized_size:314 result_ids:<71c3b949750c887e466422007cdd1a6a9f3449e3bacd43307e361e84d76fe37b> data:<130994550772:/* unknown wire type 7 */ 1642:/* unknown wire type 7 */ 10:17681930801800169409 159728:7652 9:4897805654558278394 9:/* unexpected EOF */ >min_time_ms:1506587706078 max_time_ms:1506588006078 
----------btm inputs:&{1 [123 125] asset_id:</* proto: integer overflow */ >amount:1470000000000000000  [] []}
----------issue inputs:&{1 [] 0xc4204c4120 [] []}
xprv_asset:a89d5d5fa68af8ca8408d405db180bc5b2652d7f34bca753531861be3c1cbb6216a296e1fd0f59c7215b267f24f4620e30f7c3281fa4c7a5f29ad53d8d7822bb
sign tpl:{version:1 serialized_size:314 result_ids:<71c3b949750c887e466422007cdd1a6a9f3449e3bacd43307e361e84d76fe37b> data:<130994550772:/* unknown wire type 7 */ 1642:/* unknown wire type 7 */ 10:17681930801800169409 159728:7652 9:4897805654558278394 9:/* unexpected EOF */ >min_time_ms:1506587706078 max_time_ms:1506588006078  [0xc4204c9060 0xc4204c91e0] true false}
sign tpl's SigningInstructions:&{0 [0xc420010670]}
SigningInstructions's SignatureWitnesses:&{0 [] [32 254 83 225 251 124 27 13 126 32 0 93 132 151 197 166 125 64 222 168 154 133 219 122 187 130 169 176 160 166 8 49 145 174 135] []}
submit transaction:[map[id:cc4313fbae424bb945029adef193154f34de324316036e510bcc751d0013ccb7]]
```

Query the assets on-chain:
```bash
$ ./bytomcli list-balances
0 ----- map[<gold_asset_id>:<gold_asset_amount>]
```

### Multiple node

Get the submodule depenency for the two-node test:

```bash
$ git submodule update --init --recursive
```

Create the first node `bytom0` and second node `bytom1`:

```bash
$ cd cmd/bytom/2node-test
$ ./test.sh bytom0  # Start the first node
$ ./test.sh bytom1  # Start the second node
```

Then we have two nodes:

```bash
$ curl -X POST --data '{"jsonrpc":"2.0", "method": "net_info", "params":[], "id":"67"}' http://127.0.0.1:46657
```

If everything goes well, we'll see the following response:

```bash
{
  "jsonrpc": "2.0",
  "id": "67",
  "result": {
    "listening": true,
    "listeners": [
      "Listener(@192.168.199.178:3332)"
    ],
    "peers": [
      {
        "node_info": {
          "pub_key": "03571A5CE8B35E95E2357DB2823E9EB76EB42D5CCC5F8E68315388832878C011",
          "moniker": "anonymous",
          "network": "chain0",
          "remote_addr": "127.0.0.1:51058",
          "listen_addr": "192.168.199.178:3333",
          "version": "0.1.0",
          "other": [
            "wire_version=0.6.2",
            "p2p_version=0.5.0",
            "rpc_addr=tcp://0.0.0.0:46658"
          ]
        },
......
```

## Contributing

Thank you for considering to help out with the source code! Any contributions are highly appreciated, and we are grateful for even the smallest of fixes!

If you run into an issue, feel free to [file one](https://github.com/Bytom/bytom/issues/) in this repository. We are glad to help!

## License

[AGPL v3](./LICENSE)
