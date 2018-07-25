Bytom
=====
[![Build Status](https://travis-ci.org/Bytom/bytom.svg)](https://travis-ci.org/Bytom/bytom)
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
  * [Initialize](#initialize)
  * [launch](#launch)
    * [Dashboard](#dashboard)
  * [Create key](#create-key)
  * [Create account](#create-account)
    * [Multi-signature account](#multi-signature-account)
  * [Create asset](#create-asset)
    * [Multi-signature asset](#multi-signature-asset)
  * [Sending transaction](#sending-transaction)
    * [Issue](#issue)
      * [`build-transaction`](#build-transaction)
      * [`sign-submit-transaction`](#sign-submit-transaction)
    * [Spend](#spend)
      * [`create-account-receiver`](#create-account-receiver)
      * [`build-transaction`](#build-transaction-1)
      * [`sign-submit-transaction`](#sign-submit-transaction-1)
    * [Transfer BTM](#transfer-btm)
* [Running Bytom in Docker](#running-bytom-in-docker)
* [Contributing](#contributing)
* [License](#license)

<!-- vim-markdown-toc -->

## What is Bytom?

Bytom is software designed to operate and connect to highly scalable blockchain networks confirming to the Bytom Blockchain Protocol, which allows partipicants to define, issue and transfer digitial assets on a multi-asset shared ledger. Please refer to the [White Paper](https://github.com/Bytom/wiki/blob/master/White-Paper/%E6%AF%94%E5%8E%9F%E9%93%BE%E6%8A%80%E6%9C%AF%E7%99%BD%E7%9A%AE%E4%B9%A6-%E8%8B%B1%E6%96%87%E7%89%88.md) for more details.

In the current state `bytom` is able to:

- Manage key, account as well as asset
- Send transactions, i.e., issue, spend and retire asset

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
$ git clone https://github.com/Bytom/bytom.git $GOPATH/src/github.com/bytom
```

#### Build

``` bash
$ cd $GOPATH/src/github.com/bytom
$ make bytomd    # build bytomd
$ make bytomcli  # build bytomcli
```

When successfully building the project, the `bytom` and `bytomcli` binary should be present in `cmd/bytomd` and `cmd/bytomcli` directory, respectively.

__simd feature:__

You could enable the _simd_ feature to speed up the _PoW_ verification (e.g., during mining and block verification) by simply:
```
bytomd node --simd.enable
```

To enable this feature you will need to compile from the source code by yourself, and `make bytomd-simd`. 

What is more,

+ if you are using _Mac_, please make sure _llvm_ is installed by `brew install llvm`.
+ if you are using _Windows_, please make sure _mingw-w64_ is installed and set up the _PATH_ environment variable accordingly.

## Example

Currently, bytom is still in active development and a ton of work needs to be done, but we also provide the following content for these eager to do something with `bytom`. This section won't cover all the commands of `bytomd` and `bytomcli` at length, for more information, please the help of every command, e.g., `bytomcli help`.

### Initialize

First of all, initialize the node:

```bash
$ cd ./cmd/bytomd
$ ./bytomd init --chain_id mainnet
```

There are three options for the flag `--chain_id`:

- `mainnet`: connect to the mainnet.
- `testnet`: connect to the testnet wisdom.
- `solonet`: standalone mode.

After that, you'll see `config.toml` generated, then launch the node.

### launch

``` bash
$ ./bytomd node --mining
```

available flags for `bytomd node`:

```
      --auth.disable                Disable rpc access authenticate
      --chain_id string             Select network type
  -h, --help                        help for node
      --mining                      Enable mining
      --p2p.dial_timeout int        Set dial timeout (default 3)
      --p2p.handshake_timeout int   Set handshake timeout (default 30)
      --p2p.laddr string            Node listen address.
      --p2p.max_num_peers int       Set max num peers (default 50)
      --p2p.pex                     Enable Peer-Exchange  (default true)
      --p2p.seeds string            Comma delimited host:port seed nodes
      --p2p.skip_upnp               Skip UPNP configuration
      --prof_laddr string           Use http to profile bytomd programs
      --vault_mode                  Run in the offline enviroment
      --wallet.disable              Disable wallet
      --wallet.rescan               Rescan wallet
      --web.closed                  Lanch web browser or not
```

Given the `bytomd` node is running, the general workflow is as follows:

- create key, then you can create account and asset.
- send transaction, i.e., build, sign and submit transaction.
- query all kinds of information, let's say, avaliable key, account, key, balances, transactions, etc.

#### Dashboard

Access the dashboard:

```bash
$ open http://localhost:9888/
```

### Create key

You can create a key with the following command:

```bash
$ ./bytomcli create-key alice 123
{
  "alias": "alice",
  "xpub": "d91df216da6c5641ef454c8da1e56362f86ed80d8b8fc26ab77746e1b92d6d3aa8023fe300e4c74036460d01349e4eb25cb3d7379bad855879017bc1c76165bb"
}
```

list the keys:

```bash
$ ./bytomcli list-keys
```

### Create account

Create an account named `alice`:

```bash
$ ./bytomcli create-account alice d91df216da6c5641ef454c8da1e56362f86ed80d8b8fc26ab77746e1b92d6d3aa8023fe300e4c74036460d01349e4eb25cb3d7379bad855879017bc1c76165bb
{
  "alias": "alice",
  "id": "0CIT3OI100A04",
  "key_index": 1,
  "quorum": 1,
  "xpubs": [
    "d91df216da6c5641ef454c8da1e56362f86ed80d8b8fc26ab77746e1b92d6d3aa8023fe300e4c74036460d01349e4eb25cb3d7379bad855879017bc1c76165bb"
  ]
}
```

Check out the new created account:

```bash
$ ./bytomcli list-accounts
```

#### Multi-signature account

```bash
$ ./bytomcli list-keys
0 :
{
  "alias": "default",
  "xpub": "336150c3a63411f597d94aa26fe714a348b2e93f2c303d526bb225e5804466e366e58cff81fddaed7879586b92132d63c68b419856f85ca06abfc490a9990c38"
}
1 :
{
  "alias": "alice",
  "xpub": "d91df216da6c5641ef454c8da1e56362f86ed80d8b8fc26ab77746e1b92d6d3aa8023fe300e4c74036460d01349e4eb25cb3d7379bad855879017bc1c76165bb"
}
2 :
{
  "alias": "bob",
  "xpub": "cb6057b683c5a341ea29e02ec5bb1e53691eb3b14c285138175d42b07d0551798977fc50203bde1dc2827a07f6f26237fa8ec3c6a2ef272ed80f9211f9c6ac64"
}
```

```bash
$ ./bytomcli create-account multi_account d91df216da6c5641ef454c8da1e56362f86ed80d8b8fc26ab77746e1b92d6d3aa8023fe300e4c74036460d01349e4eb25cb3d7379bad855879017bc1c76165bb cb6057b683c5a341ea29e02ec5bb1e53691eb3b14c285138175d42b07d0551798977fc50203bde1dc2827a07f6f26237fa8ec3c6a2ef272ed80f9211f9c6ac64 -q 2
{
  "alias": "multi_account",
  "id": "0CIT6J0QG0A06",
  "key_index": 1,
  "quorum": 2,
  "xpubs": [
    "cb6057b683c5a341ea29e02ec5bb1e53691eb3b14c285138175d42b07d0551798977fc50203bde1dc2827a07f6f26237fa8ec3c6a2ef272ed80f9211f9c6ac64",
    "d91df216da6c5641ef454c8da1e56362f86ed80d8b8fc26ab77746e1b92d6d3aa8023fe300e4c74036460d01349e4eb25cb3d7379bad855879017bc1c76165bb"
  ]
}
```

### Create asset

Create an asset named `GOLD`:

```bash
$ ./bytomcli create-asset GOLD d91df216da6c5641ef454c8da1e56362f86ed80d8b8fc26ab77746e1b92d6d3aa8023fe300e4c74036460d01349e4eb25cb3d7379bad855879017bc1c76165bb
{
  "alias": "GOLD",
  "definition": {},
  "id": "43e89d8a5d8f4bb2fcd92621ace61d476c8237075ada6a9e80db931bbdb6c397",
  "issuance_program": "ae2035b53ed466a40e3d1073ffdcf1c7d0c4ffc439391f9ef5999b365ea467c96a3c5151ad",
  "keys": [
    {
      "asset_derivation_path": [
        "000100000000000000"
      ],
      "asset_pubkey": "35b53ed466a40e3d1073ffdcf1c7d0c4ffc439391f9ef5999b365ea467c96a3c0e8b6d5a67bd5fc66d7a19d4754df6de6cbf3b40fc6b02a75f140d77a1dbcda8",
      "root_xpub": "d91df216da6c5641ef454c8da1e56362f86ed80d8b8fc26ab77746e1b92d6d3aa8023fe300e4c74036460d01349e4eb25cb3d7379bad855879017bc1c76165bb"
    }
  ],
  "quorum": 1
}
```

Check out the new created asset:

```bash
$ ./bytomcli list-assets
```

#### Multi-signature asset

```bash
$ ./bytomcli create-asset SILVER d91df216da6c5641ef454c8da1e56362f86ed80d8b8fc26ab77746e1b92d6d3aa8023fe300e4c74036460d01349e4eb25cb3d7379bad855879017bc1c76165bb cb6057b683c5a341ea29e02ec5bb1e53691eb3b14c285138175d42b07d0551798977fc50203bde1dc2827a07f6f26237fa8ec3c6a2ef272ed80f9211f9c6ac64
{
  "alias": "SILVER",
  "definition": {},
  "id": "50cd5b0ecfdfe388fc0bc6b74df6becd94f1e52e1913a449a473bba87745fe30",
  "issuance_program": "ae20936178877c2a3a2df9c3733e2ac6f7e99477314c54e214348abb0d4cbdace810208d064a125805493cb99be9c19ed6a356c97753fdfe52fade337fa598c3c59b8c5152ad",
  "keys": [
    {
      "asset_derivation_path": [
        "000200000000000000"
      ],
      "asset_pubkey": "936178877c2a3a2df9c3733e2ac6f7e99477314c54e214348abb0d4cbdace810f9ef97f8edd43707069426ff1ac1d2fe96d23c5e005ac5cb52be2a7de82d2a92",
      "root_xpub": "cb6057b683c5a341ea29e02ec5bb1e53691eb3b14c285138175d42b07d0551798977fc50203bde1dc2827a07f6f26237fa8ec3c6a2ef272ed80f9211f9c6ac64"
    },
    {
      "asset_derivation_path": [
        "000200000000000000"
      ],
      "asset_pubkey": "8d064a125805493cb99be9c19ed6a356c97753fdfe52fade337fa598c3c59b8c4eb0b479d566c0efe72ab69f2273ab4b6a6dce9e1e3657839698be6c53e9d04a",
      "root_xpub": "d91df216da6c5641ef454c8da1e56362f86ed80d8b8fc26ab77746e1b92d6d3aa8023fe300e4c74036460d01349e4eb25cb3d7379bad855879017bc1c76165bb"
    }
  ],
  "quorum": 1
}
```

### Sending transaction

Every asset-related action is trigger via sending a transaction, which requires two steps to complete, i.e., `build-transaction` and `sign-submit-transaction`. Don't forget to enable `--mining`.

#### Issue

##### `build-transaction`

Let's say, issue 10000 GOLD to alice:
```bash
$ ./bytomcli create-account-receiver alice
{
  "address": "bm1q7s24rra4j05yhec9chry2c9trd6qa8gjr6cue3",
  "control_program": "0014f415518fb593e84be705c5c64560ab1b740e9d12"
}
```

```bash
$ ./bytomcli build-transaction -t issue alice GOLD 10000 -a bm1q7s24rra4j05yhec9chry2c9trd6qa8gjr6cue3 --alias
Template Type: issue
{"allow_additional_actions":false,"raw_transaction":"070100020160015ee56e4d688e98160067fa25be520d3a5a90e63838b5ff6363997c7c9b962796daffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8094ebdc030501160014f415518fb593e84be705c5c64560ab1b740e9d120100012c000887c3aad437888d0143e89d8a5d8f4bb2fcd92621ace61d476c8237075ada6a9e80db931bbdb6c397904e29000125ae2035b53ed466a40e3d1073ffdcf1c7d0c4ffc439391f9ef5999b365ea467c96a3c5151ad0002013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80baa6d30301160014c0c4fb15dd132be34e6dcaea3c38df1869ebcdbf00013a43e89d8a5d8f4bb2fcd92621ace61d476c8237075ada6a9e80db931bbdb6c397904e01160014f415518fb593e84be705c5c64560ab1b740e9d1200","signing_instructions":[{"position":0,"witness_components":[{"keys":[{"derivation_path":["010100000000000000","0200000000000000"],"xpub":"d91df216da6c5641ef454c8da1e56362f86ed80d8b8fc26ab77746e1b92d6d3aa8023fe300e4c74036460d01349e4eb25cb3d7379bad855879017bc1c76165bb"}],"quorum":1,"signatures":null,"type":"raw_tx_signature"},{"type":"data","value":"4dc1e7f4e9710378cb80f180e159005a5ca8934d68df6136b782fe08d23d7b0a"}]},{"position":1,"witness_components":[{"keys":[{"derivation_path":["000100000000000000"],"xpub":"d91df216da6c5641ef454c8da1e56362f86ed80d8b8fc26ab77746e1b92d6d3aa8023fe300e4c74036460d01349e4eb25cb3d7379bad855879017bc1c76165bb"}],"quorum":1,"signatures":null,"type":"raw_tx_signature"}]}]}
```

The response of `build-transaction` will be used in the following `sign-submit-transaction` command.

##### `sign-submit-transaction`

```bash
$ ./bytomcli sign-submit-transaction '{"allow_additional_actions":false,"raw_transaction":"070100020160015ee56e4d688e98160067fa25be520d3a5a90e63838b5ff6363997c7c9b962796daffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8094ebdc030501160014f415518fb593e84be705c5c64560ab1b740e9d120100012c000887c3aad437888d0143e89d8a5d8f4bb2fcd92621ace61d476c8237075ada6a9e80db931bbdb6c397904e29000125ae2035b53ed466a40e3d1073ffdcf1c7d0c4ffc439391f9ef5999b365ea467c96a3c5151ad0002013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80baa6d30301160014c0c4fb15dd132be34e6dcaea3c38df1869ebcdbf00013a43e89d8a5d8f4bb2fcd92621ace61d476c8237075ada6a9e80db931bbdb6c397904e01160014f415518fb593e84be705c5c64560ab1b740e9d1200","signing_instructions":[{"position":0,"witness_components":[{"keys":[{"derivation_path":["010100000000000000","0200000000000000"],"xpub":"d91df216da6c5641ef454c8da1e56362f86ed80d8b8fc26ab77746e1b92d6d3aa8023fe300e4c74036460d01349e4eb25cb3d7379bad855879017bc1c76165bb"}],"quorum":1,"signatures":null,"type":"raw_tx_signature"},{"type":"data","value":"4dc1e7f4e9710378cb80f180e159005a5ca8934d68df6136b782fe08d23d7b0a"}]},{"position":1,"witness_components":[{"keys":[{"derivation_path":["000100000000000000"],"xpub":"d91df216da6c5641ef454c8da1e56362f86ed80d8b8fc26ab77746e1b92d6d3aa8023fe300e4c74036460d01349e4eb25cb3d7379bad855879017bc1c76165bb"}],"quorum":1,"signatures":null,"type":"raw_tx_signature"}]}]}' -p 123 

{
  "tx_id": "62fe1716f2b8fe012578a3a8f6b02bd2fa548fef5bba792ea78673e07b38198d"
}
```

When the transaction is on-chain, query the balances:

```bash
# alice should have 10000 GOLD now
$ ./bytomcli list-balances
0 :
{
  "account_alias": "default",
  "account_id": "0CIT2D2O00A02",
  "amount": 2098770000000,
  "asset_alias": "BTM",
  "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
}
1 :
{
  "account_alias": "alice",
  "account_id": "0CIT3OI100A04",
  "amount": 10000,
  "asset_alias": "GOLD",
  "asset_id": "43e89d8a5d8f4bb2fcd92621ace61d476c8237075ada6a9e80db931bbdb6c397"
}
2 :
{
  "account_alias": "alice",
  "account_id": "0CIT3OI100A04",
  "amount": 4980000000,
  "asset_alias": "BTM",
  "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
}
```

#### Spend

Alice pays Bob `<payment_amount>`, e.g., 1000, `GOLD`:

##### `create-account-receiver`

Before you transfer an asset to another account, you have to know his `control_program`. This means the receiver needs to send you his `control_program` first.

```bash
$ ./bytomcli create-account-receiver bob
{
  "address": "bm1qhurffd3gqkc3pttqfqpsz8w04jnl5z97798tpt",
  "control_program": "0014bf0694b62805b110ad604803011dcfaca7fa08be"
}
```

##### `build-transaction`

```bash
# ./bytomcli build-transaction -t spend <sender_account> <asset> <amount> --alias -r <receiver_control_program>
$ ./bytomcli build-transaction -t spend alice GOLD 1000 --alias -r 0014bf0694b62805b110ad604803011dcfaca7fa08be
Template Type: spend
{"allow_additional_actions":false,"raw_transaction":"070100020160015eba7bbf304b94623b1efb414845f701e64e6c63011984e65dcf214d560f505e28ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80baa6d3030001160014c0c4fb15dd132be34e6dcaea3c38df1869ebcdbf0100015d015bba7bbf304b94623b1efb414845f701e64e6c63011984e65dcf214d560f505e2843e89d8a5d8f4bb2fcd92621ace61d476c8237075ada6a9e80db931bbdb6c397904e0101160014f415518fb593e84be705c5c64560ab1b740e9d12010003013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80e0e1c903011600144457a5103dc682327fe2ef662f812dce3f97ffae00013a43e89d8a5d8f4bb2fcd92621ace61d476c8237075ada6a9e80db931bbdb6c397a846011600147460e50dc0b0313d22796c899c28f795872b229600013a43e89d8a5d8f4bb2fcd92621ace61d476c8237075ada6a9e80db931bbdb6c397e80701160014bf0694b62805b110ad604803011dcfaca7fa08be00","signing_instructions":[{"position":0,"witness_components":[{"keys":[{"derivation_path":["010100000000000000","0300000000000000"],"xpub":"d91df216da6c5641ef454c8da1e56362f86ed80d8b8fc26ab77746e1b92d6d3aa8023fe300e4c74036460d01349e4eb25cb3d7379bad855879017bc1c76165bb"}],"quorum":1,"signatures":null,"type":"raw_tx_signature"},{"type":"data","value":"eef67105644a1f146ba313a7ea98ceac2534a9f4cf1aa3775b726476db9c3dc0"}]},{"position":1,"witness_components":[{"keys":[{"derivation_path":["010100000000000000","0200000000000000"],"xpub":"d91df216da6c5641ef454c8da1e56362f86ed80d8b8fc26ab77746e1b92d6d3aa8023fe300e4c74036460d01349e4eb25cb3d7379bad855879017bc1c76165bb"}],"quorum":1,"signatures":null,"type":"raw_tx_signature"},{"type":"data","value":"4dc1e7f4e9710378cb80f180e159005a5ca8934d68df6136b782fe08d23d7b0a"}]}]}
```

##### `sign-submit-transaction`

```bash
$ ./bytomcli sign-submit-transaction '{"allow_additional_actions":false,"raw_transaction":"070100020160015eba7bbf304b94623b1efb414845f701e64e6c63011984e65dcf214d560f505e28ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80baa6d3030001160014c0c4fb15dd132be34e6dcaea3c38df1869ebcdbf0100015d015bba7bbf304b94623b1efb414845f701e64e6c63011984e65dcf214d560f505e2843e89d8a5d8f4bb2fcd92621ace61d476c8237075ada6a9e80db931bbdb6c397904e0101160014f415518fb593e84be705c5c64560ab1b740e9d12010003013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80e0e1c903011600144457a5103dc682327fe2ef662f812dce3f97ffae00013a43e89d8a5d8f4bb2fcd92621ace61d476c8237075ada6a9e80db931bbdb6c397a846011600147460e50dc0b0313d22796c899c28f795872b229600013a43e89d8a5d8f4bb2fcd92621ace61d476c8237075ada6a9e80db931bbdb6c397e80701160014bf0694b62805b110ad604803011dcfaca7fa08be00","signing_instructions":[{"position":0,"witness_components":[{"keys":[{"derivation_path":["010100000000000000","0300000000000000"],"xpub":"d91df216da6c5641ef454c8da1e56362f86ed80d8b8fc26ab77746e1b92d6d3aa8023fe300e4c74036460d01349e4eb25cb3d7379bad855879017bc1c76165bb"}],"quorum":1,"signatures":null,"type":"raw_tx_signature"},{"type":"data","value":"eef67105644a1f146ba313a7ea98ceac2534a9f4cf1aa3775b726476db9c3dc0"}]},{"position":1,"witness_components":[{"keys":[{"derivation_path":["010100000000000000","0200000000000000"],"xpub":"d91df216da6c5641ef454c8da1e56362f86ed80d8b8fc26ab77746e1b92d6d3aa8023fe300e4c74036460d01349e4eb25cb3d7379bad855879017bc1c76165bb"}],"quorum":1,"signatures":null,"type":"raw_tx_signature"},{"type":"data","value":"4dc1e7f4e9710378cb80f180e159005a5ca8934d68df6136b782fe08d23d7b0a"}]}]}'


{
  "txid": "3bf01bd21d7bbf3eb1c6e27e5646be1c4a7bd5602125eebf28f430246b08c226"
}
```

```bash
$./bytomcli list-balances
0 :
{
  "account_alias": "default",
  "account_id": "0CIT2D2O00A02",
  "amount": 2758790000000,
  "asset_alias": "BTM",
  "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
}
1 :
{
  "account_alias": "alice",
  "account_id": "0CIT3OI100A04",
  "amount": 9000,
  "asset_alias": "GOLD",
  "asset_id": "43e89d8a5d8f4bb2fcd92621ace61d476c8237075ada6a9e80db931bbdb6c397"
}
2 :
{
  "account_alias": "alice",
  "account_id": "0CIT3OI100A04",
  "amount": 4960000000,
  "asset_alias": "BTM",
  "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
}
3 :
{
  "account_alias": "bob",
  "account_id": "0CITLN2KG0A08",
  "amount": 1000,
  "asset_alias": "GOLD",
  "asset_id": "43e89d8a5d8f4bb2fcd92621ace61d476c8237075ada6a9e80db931bbdb6c397"
}
```

#### Transfer BTM

```bash
# ./bytomcli build-transaction -t spend <sender_account> <asset> <amount> --alias -r <receiver_control_program>
$ ./bytomcli build-transaction -t spend alice BTM 1000000000 --alias -r 0014bf0694b62805b110ad604803011dcfaca7fa08be
Template Type: spend
{"allow_additional_actions":false,"raw_transaction":"070100020160015ee56e4d688e98160067fa25be520d3a5a90e63838b5ff6363997c7c9b962796daffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8094ebdc030201160014f415518fb593e84be705c5c64560ab1b740e9d1201000160015e5685c051b6a7f1631e0505ea99a61997c1b8322148321eb864435c8516a5a8eaffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80e0e1c90300011600144457a5103dc682327fe2ef662f812dce3f97ffae010002013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80869dc003011600146026f938bbcf6b6b90eb9d8fbb8c1725e0ae3d0b00013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8094ebdc0301160014bf0694b62805b110ad604803011dcfaca7fa08be00","signing_instructions":[{"position":0,"witness_components":[{"keys":[{"derivation_path":["010100000000000000","0200000000000000"],"xpub":"d91df216da6c5641ef454c8da1e56362f86ed80d8b8fc26ab77746e1b92d6d3aa8023fe300e4c74036460d01349e4eb25cb3d7379bad855879017bc1c76165bb"}],"quorum":1,"signatures":null,"type":"raw_tx_signature"},{"type":"data","value":"4dc1e7f4e9710378cb80f180e159005a5ca8934d68df6136b782fe08d23d7b0a"}]},{"position":1,"witness_components":[{"keys":[{"derivation_path":["010100000000000000","0400000000000000"],"xpub":"d91df216da6c5641ef454c8da1e56362f86ed80d8b8fc26ab77746e1b92d6d3aa8023fe300e4c74036460d01349e4eb25cb3d7379bad855879017bc1c76165bb"}],"quorum":1,"signatures":null,"type":"raw_tx_signature"},{"type":"data","value":"2cfbc821a459ef53b9ee4a170b735e54f968f239bf9155712f4b88256f97d54a"}]}]}
```

```bash
$ ./bytomcli sign-submit-transaction '{"allow_additional_actions":false,"raw_transaction":"070100020160015ee56e4d688e98160067fa25be520d3a5a90e63838b5ff6363997c7c9b962796daffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8094ebdc030201160014f415518fb593e84be705c5c64560ab1b740e9d1201000160015e5685c051b6a7f1631e0505ea99a61997c1b8322148321eb864435c8516a5a8eaffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80e0e1c90300011600144457a5103dc682327fe2ef662f812dce3f97ffae010002013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80869dc003011600146026f938bbcf6b6b90eb9d8fbb8c1725e0ae3d0b00013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8094ebdc0301160014bf0694b62805b110ad604803011dcfaca7fa08be00","signing_instructions":[{"position":0,"witness_components":[{"keys":[{"derivation_path":["010100000000000000","0200000000000000"],"xpub":"d91df216da6c5641ef454c8da1e56362f86ed80d8b8fc26ab77746e1b92d6d3aa8023fe300e4c74036460d01349e4eb25cb3d7379bad855879017bc1c76165bb"}],"quorum":1,"signatures":null,"type":"raw_tx_signature"},{"type":"data","value":"4dc1e7f4e9710378cb80f180e159005a5ca8934d68df6136b782fe08d23d7b0a"}]},{"position":1,"witness_components":[{"keys":[{"derivation_path":["010100000000000000","0400000000000000"],"xpub":"d91df216da6c5641ef454c8da1e56362f86ed80d8b8fc26ab77746e1b92d6d3aa8023fe300e4c74036460d01349e4eb25cb3d7379bad855879017bc1c76165bb"}],"quorum":1,"signatures":null,"type":"raw_tx_signature"},{"type":"data","value":"2cfbc821a459ef53b9ee4a170b735e54f968f239bf9155712f4b88256f97d54a"}]}]}' -p 123


{
  "tx_id": "1c84c8d8dae7675114f9de970e221ec9e07e1787f6e72ac6e95a03fced7d22cb"
}
```

```bash
$ ./bytomcli list-balances
0 :
{
  "account_alias": "default",
  "account_id": "0CIT2D2O00A02",
  "amount": 3212560000000,
  "asset_alias": "BTM",
  "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
}
1 :
{
  "account_alias": "alice",
  "account_id": "0CIT3OI100A04",
  "amount": 9000,
  "asset_alias": "GOLD",
  "asset_id": "43e89d8a5d8f4bb2fcd92621ace61d476c8237075ada6a9e80db931bbdb6c397"
}
2 :
{
  "account_alias": "alice",
  "account_id": "0CIT3OI100A04",
  "amount": 3940000000,
  "asset_alias": "BTM",
  "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
}
3 :
{
  "account_alias": "bob",
  "account_id": "0CITLN2KG0A08",
  "amount": 1000,
  "asset_alias": "GOLD",
  "asset_id": "43e89d8a5d8f4bb2fcd92621ace61d476c8237075ada6a9e80db931bbdb6c397"
}
4 :
{
  "account_alias": "bob",
  "account_id": "0CITLN2KG0A08",
  "amount": 1000000000,
  "asset_alias": "BTM",
  "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
}
```

## Running Bytom in Docker

Ensure your [Docker](https://www.docker.com/) version is 17.05 or higher.

```bash
$ docker build -t bytom .
```

For the usage please refer to [here](https://github.com/Bytom/bytom/wiki/Running-in-Docker).

## Contributing

Thank you for considering helping out with the source code! Any contributions are highly appreciated, and we are grateful for even the smallest of fixes!

If you run into an issue, feel free to [file one](https://github.com/Bytom/bytom/issues/) in this repository. We are glad to help!

## License

[AGPL v3](./LICENSE)
