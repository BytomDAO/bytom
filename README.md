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
  * [Initialize and launch](#initialize-and-launch)
  * [Create key](#create-key)
  * [Create account](#create-account)
    * [Multi-signature account](#multi-signature-account)
  * [Create asset](#create-asset)
    * [Multi-signature asset](#multi-signature-asset)
  * [Sending transaction](#sending-transaction)
    * [Issue](#issue)
      * [`build transaction`](#build-transaction)
      * [`sign-submit-transaction`](#sign-submit-transaction)
    * [Spend](#spend)
      * [`build-transaction`](#build-transaction-1)
      * [`sign-submit-transaction`](#sign-submit-transaction-1)
    * [Transfer BTM](#transfer-btm)
  * [Multiple node](#multiple-node)
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
$ git clone https://github.com/Bytom/bytom $GOPATH/src/github.com/bytom
```

#### Build

``` bash
$ cd $GOPATH/src/github.com/bytom
$ make bytomd    # build bytomd
$ make bytomcli  # build bytomcli
```

When successfully building the project, the `bytom` and `bytomcli` binary should be present in `cmd/bytomd` and `cmd/bytomcli` directory, respectively.

## Example

Currently, bytom is still in active development and a ton of work needs to be done, but we also provide the following content for these eager to do something with `bytom`. This section won't cover all the commands of `bytomd` and `bytomcli` at length, for more information, please the help of every command, e.g., `bytomcli help`.

### Initialize and launch

First of all, initialize the node:

```bash
$ cd ./cmd/bytomd
$ ./bytomd init --chain_id testnet
```

There are two options for the flag `--chain_id`:

- `testnet`: connect to the testnet.
- `mainnet`: standalone mode.

After that, you'll see `.bytomd` generated in current directory, then launch the node:

``` bash
$ ./bytomd node --wallet.enable
```

available flags for `bytomd node`:

- `--wallet.enable`
- `--mining`

Given the `bytom` node is running, the general workflow is as follows:

- create key, then you can create account and asset.
- send transaction, i.e., build, sign and submit transaction.
- query all kinds of information, let's say, avaliable key, account, key, balances, transactions, etc.

### Create key

With `--wallet.enable`, you can create a key:

```bash
$ ./bytomcli create-key alice 123
{
  "alias": "alice",
  "file": "/Users/xlc/go/src/github.com/bytom/cmd/bytomd/.bytomd/keystore/UTC--2018-01-02T07-40-49.900440000Z--b6700ba3-befd-4750-9e58-df91eba15e25",
  "xpub": "674b0d709c5c2f6eb49e5f205c4393e91f1fa745ff62954b67be546925480af57f12a20b2f963a489a71c61b40d5fa08e8c522c8f2d49eaa1213dcb0a4350afb"
}
```

list the keys:

```bash
$ ./bytomcli list-keys
```

### Create account

Create an account named `alice`:

```bash
$ ./bytomcli create-account alice 674b0d709c5c2f6eb49e5f205c4393e91f1fa745ff62954b67be546925480af57f12a20b2f963a489a71c61b40d5fa08e8c522c8f2d49eaa1213dcb0a4350afb
{
  "alias": "alice",
  "id": "08FTNE7000A02",
  "keys": [
    {
      "account_derivation_path": [
        "010100000000000000"
      ],
      "account_xpub": "f78fa93f009b7baba1b42dca21d06a7902c41ce5fd8f16db625ca41b95a60116cb3bf87c7a098fad9172db5e7481cd4225bb6032b56a310462677fd030243e48",
      "root_xpub": "674b0d709c5c2f6eb49e5f205c4393e91f1fa745ff62954b67be546925480af57f12a20b2f963a489a71c61b40d5fa08e8c522c8f2d49eaa1213dcb0a4350afb"
    }
  ],
  "quorum": 1,
  "tags": null
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
  "alias": "alice",
  "file": "/Users/xlc/go/src/github.com/bytom/cmd/bytomd/.bytomd/keystore/UTC--2018-01-02T07-40-49.900440000Z--b6700ba3-befd-4750-9e58-df91eba15e25",
  "xpub": "674b0d709c5c2f6eb49e5f205c4393e91f1fa745ff62954b67be546925480af57f12a20b2f963a489a71c61b40d5fa08e8c522c8f2d49eaa1213dcb0a4350afb"
}
1 :
{
  "alias": "bob",
  "file": "/Users/xlc/go/src/github.com/bytom/cmd/bytomd/.bytomd/keystore/UTC--2018-01-02T07-41-35.458581000Z--7115fa1c-a5cd-4374-a747-75d038c402a9",
  "xpub": "db0abf717c37fb01cb89d06a01d087f24a15d07e99d297b3c02642c46166e2103a7525ec85a2c8050a02b5864fd295ee933c09ca341b296a55e0f804ecb9d9a3"
}
```

```bash
$ ./bytomcli create-account multi_account 674b0d709c5c2f6eb49e5f205c4393e91f1fa745ff62954b67be546925480af57f12a20b2f963a489a71c61b40d5fa08e8c522c8f2d49eaa1213dcb0a4350afb db0abf717c37fb01cb89d06a01d087f24a15d07e99d297b3c02642c46166e2103a7525ec85a2c8050a02b5864fd295ee933c09ca341b296a55e0f804ecb9d9a3 -q 2
{
  "alias": "multi_account",
  "id": "08FR9JDE00A02",
  "keys": [
    {
      "account_derivation_path": [
        "010100000000000000"
      ],
      "account_xpub": "f78fa93f009b7baba1b42dca21d06a7902c41ce5fd8f16db625ca41b95a60116cb3bf87c7a098fad9172db5e7481cd4225bb6032b56a310462677fd030243e48",
      "root_xpub": "674b0d709c5c2f6eb49e5f205c4393e91f1fa745ff62954b67be546925480af57f12a20b2f963a489a71c61b40d5fa08e8c522c8f2d49eaa1213dcb0a4350afb"
    },
    {
      "account_derivation_path": [
        "010100000000000000"
      ],
      "account_xpub": "b29be450d0b5b1e233bfa61cc4312c64287a9179d2981510c660fcb4e978fbb8a1d4e4c00882f77b492733e477c19dc2d617ff34216821106078c05ef888c274",
      "root_xpub": "db0abf717c37fb01cb89d06a01d087f24a15d07e99d297b3c02642c46166e2103a7525ec85a2c8050a02b5864fd295ee933c09ca341b296a55e0f804ecb9d9a3"
    }
  ],
  "quorum": 2,
  "tags": null
}
```

### Create asset

Create an asset named `gold`:

```bash
$ ./bytomcli create-asset gold 674b0d709c5c2f6eb49e5f205c4393e91f1fa745ff62954b67be546925480af57f12a20b2f963a489a71c61b40d5fa08e8c522c8f2d49eaa1213dcb0a4350afb
{
  "alias": "gold",
  "definition": {},
  "id": "ff34b6aea66cbd13bfaf918294e0d45fe5ac89854875aadcabb54d92bffebb27",
  "issuance_program": "766baa200677273bad4aecb9fb90f99d41dda6bb233a138e1a6ad4aeb334241171c5df075151ad696c00c0",
  "keys": [
    {
      "asset_derivation_path": [
        "000200000000000000"
      ],
      "asset_pubkey": "0677273bad4aecb9fb90f99d41dda6bb233a138e1a6ad4aeb334241171c5df07df57895f7e9c2adb909db2299dd061e60b365c0d008a1a2a14b97f56f19642d7",
      "root_xpub": "674b0d709c5c2f6eb49e5f205c4393e91f1fa745ff62954b67be546925480af57f12a20b2f963a489a71c61b40d5fa08e8c522c8f2d49eaa1213dcb0a4350afb"
    }
  ],
  "quorum": 1,
  "tags": {}
}
```

Check out the new created asset:

```bash
$ ./bytomcli list-assets
```

#### Multi-signature asset

```bash
$ ./bytomcli create-asset silver 674b0d709c5c2f6eb49e5f205c4393e91f1fa745ff62954b67be546925480af57f12a20b2f963a489a71c61b40d5fa08e8c522c8f2d49eaa1213dcb0a4350afb db0abf717c37fb01cb89d06a01d087f24a15d07e99d297b3c02642c46166e2103a7525ec85a2c8050a02b5864fd295ee933c09ca341b296a55e0f804ecb9d9a3
{
  "alias": "silver",
  "definition": {},
  "id": "6465c855881d2add50769242c0e85e4c9ef5ba8b258e6933cb41fcbb17a88b26",
  "issuance_program": "766baa207a40f1fa20149154b0c4092f151685cdf5a6bec93a8d5c9b00299730476436c6206b4bb7ee5bf3d4ab4a0c625d3dd36b42fbe21e3f3e6f972a6b0590de022960b55152ad696c00c0",
  "keys": [
    {
      "asset_derivation_path": [
        "000300000000000000"
      ],
      "asset_pubkey": "7a40f1fa20149154b0c4092f151685cdf5a6bec93a8d5c9b00299730476436c65434ac760f1b3ee35776e50ff13503079981c16263c2eaa4d3548b7cd936b533",
      "root_xpub": "674b0d709c5c2f6eb49e5f205c4393e91f1fa745ff62954b67be546925480af57f12a20b2f963a489a71c61b40d5fa08e8c522c8f2d49eaa1213dcb0a4350afb"
    },
    {
      "asset_derivation_path": [
        "000300000000000000"
      ],
      "asset_pubkey": "6b4bb7ee5bf3d4ab4a0c625d3dd36b42fbe21e3f3e6f972a6b0590de022960b5e70f2162d1a5314d140139da7947680289ad961e25266d5631b0de3732a547c3",
      "root_xpub": "db0abf717c37fb01cb89d06a01d087f24a15d07e99d297b3c02642c46166e2103a7525ec85a2c8050a02b5864fd295ee933c09ca341b296a55e0f804ecb9d9a3"
    }
  ],
  "quorum": 1,
  "tags": {}
}
```

### Sending transaction

Every asset-related action is trigger via sending a transaction, which requires two steps to complete, i.e., `build-transaction` and `sign-subit-transaction`.

#### Issue

Since the account alice and the asset `gold` are ready, issue `gold` to alice:

```bash
$ ./bytomcli create-account bob
```

##### `build transaction`

Firstly, Alice issue `<issue_amount>`, e.g., 10000, `gold`:

```bash
$ ./bytomcli sub-create-issue-tx <alice_account_id> <gold_asset_id> <issue_amount> <gold_asset_private_key> <account_private_key>
```

##### `sign-submit-transaction`

When the transaction is on-chain, query the balances:

```bash
# Alice should have 10000 gold now
$ ./bytomcli list-balances
```

#### Spend

Alice pays Bob `<payment_amount>`, e.g., 1000, `gold`:

##### `build-transaction`

- Bob creates receiver program

```bash
$./bytomcli create-account-receiver bob
```

##### `sign-submit-transaction`

responses like this:

```
responses:map[control_program:766baa207b73a71a9e8a77ace69bdd5d68fdfacc841bfdd5d6e0057a331846c24ac222e35151ad696c00c0 expires_at:2017-12-30T17:10:01.915062361+08:00]
```

- off-chain transfers receiver program to alice
- Alice builds transaction and then makes this transacion on-chain

```bash
$./bytomcli sub-control-receiver-tx <account_xprv> <account_id> <asset_id> <spend_amount> <control_program>
```

- list balance
```bash
$./bytomcli list-balances
```

#### Transfer BTM

As above, just `btm_asset_id`=ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff

```bash
$./bytomcli sub-control-receiver-tx <account_xprv> <account_id> <btm_asset_id> <spend_amount> <control_program>
```

### Multiple node

Get the submodule depenency for the two-node test:

```bash
$ git submodule update --init --recursive
```

Create the first node `bytomd0` and second node `bytomd1`:

```bash
$ cd cmd/bytomd/2node-test
$ ./test.sh bytomd0  # Start the first node
$ ./test.sh bytomd1  # Start the second node
```

Then we have two nodes:

```bash
$ ./bytomcli net-info
net-info:map[listening:true listeners:[Listener(@192.168.199.43:3332)] peers:[map[node_info:map[listen_addr:192.168.199.43:3333 version:0.1.2 other:[wire_version=0.6.2 p2p_version=0.5.0] pub_key:D6B76D1B4E9D7E4D81BA5FAAE9359302446488495A29D7E70AF84CDFEA186D66 moniker:anonymous network:bytom remote_addr:127.0.0.1:51036] is_outbound:false connection_status:map[RecvMonitor:map[Start:2017-10-30T13:45:47.18+08:00 Bytes:425130 AvgRate:27010 Progress:0 Active:true Idle:1.04e+09 Samples:42 InstRate:4591 CurRate:3540 PeakRate:114908 BytesRem:0 TimeRem:0 Duration:1.574e+10] Channels:[map[RecentlySent:5332 ID:64 SendQueueCapacity:100 SendQueueSize:0 Priority:5]] SendMonitor:map[Active:true Idle:1.24e+09 Bytes:16240 Samples:41 CurRate:125 AvgRate:1032 Progress:0 Start:2017-10-30T13:45:47.18+08:00 Duration:1.574e+10 InstRate:147 PeakRate:4375 BytesRem:0 TimeRem:0]]]]]
```

## Running Bytom in Docker

Ensure your [Docker](https://www.docker.com/) version is 17.05 or higher.

```bash
$ docker build -t bytom .
```

## Contributing

Thank you for considering to help out with the source code! Any contributions are highly appreciated, and we are grateful for even the smallest of fixes!

If you run into an issue, feel free to [file one](https://github.com/Bytom/bytom/issues/) in this repository. We are glad to help!

## License

[AGPL v3](./LICENSE)
