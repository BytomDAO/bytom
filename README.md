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
  * [Create and launch a single node](#create-and-launch-a-single-node)
    * [Create an account](#create-an-account)
    * [Create an asset](#create-an-asset)
    * [Issue an asset](#issue-an-asset)
    * [Transfer an asset](#transfer-an-asset)
    * [Transfer btm](#transfer-btm)
  * [Set up a wallet and manage the key](#set-up-a-wallet-and-manage-the-key)
  * [Multiple node](#multiple-node)
* [Running Bytom in Docker](#running-bytom-in-docker)
* [Contributing](#contributing)
* [License](#license)

<!-- vim-markdown-toc -->

## What is Bytom?

Bytom is software designed to operate and connect to highly scalable blockchain networks confirming to the Bytom Blockchain Protocol, which allows partipicants to define, issue and transfer digitial assets on a multi-asset shared ledger. Please refer to the [White Paper](https://github.com/Bytom/wiki/blob/master/White-Paper/%E6%AF%94%E5%8E%9F%E9%93%BE%E6%8A%80%E6%9C%AF%E7%99%BD%E7%9A%AE%E4%B9%A6-%E8%8B%B1%E6%96%87%E7%89%88.md) for more details.

In the current state `bytom` is able to:

- Issue assets
- Manage account as well as asset
- Spend assets

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

## Example

Currently, bytom is still in active development and a ton of work needs to be done, but we also provide the following content for these eager to do something with `bytom`.

### Create and launch a single node

When successfully building the project, the `bytom` and `bytomcli` binary should be present in `cmd/bytomd` and `cmd/bytomcli` directory, respectively. The next step is to initialize the node:

```bash
$ cd ./cmd/bytomd
$ ./bytomd init --chain_id testnet
```

After that, you'll see `.bytom` generated in current directory, then launch the single node:

``` bash
$ ./bytomd node --wallet.enable
```

Given the `bytom` node is running, the general workflow is as follows:

- create an account
- create an asset
- create/sign/submit a transaction to transfer an asset
- query the assets on-chain


#### Create an account

Create an account named `alice`:

```bash
$ ./bytomcli create-account alice
xprv:<alice_account_private_key>
responses:<create-account-responses>
account id:<alice_account_id>
```

Check out the new created account:

```bash
$ ./bytomcli list-accounts
```

#### Create an asset

Create an asset named `gold`:

```bash
$ ./bytomcli create-asset gold
xprv:<gold_asset_private_key>
xpub:<gold_asset_public_key>
responses:<create-asset-responses>
asset id:<gold_asset_id>
```

Check out the new created asset:

```bash
$ ./bytomcli list-assets
```

#### Issue an asset

Since the account alice and the asset `gold` are ready, issue `gold` to alice:

```bash
$ ./bytomcli create-account bob
```

Firstly, Alice issue `<issue_amount>`, e.g., 10000, `gold`:

```bash
$ ./bytomcli sub-create-issue-tx <alice_account_id> <gold_asset_id> <issue_amount> <gold_asset_private_key> <account_private_key>
```

When the transaction is on-chain, query the balances:

```bash
# Alice should have 10000 gold now
$ ./bytomcli list-balances
```

#### Transfer an asset

Alice pays Bob `<payment_amount>`, e.g., 1000, `gold`:
- Bob creates receiver program
```bash
$./bytomcli create-account-receiver bob 
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

#### Transfer btm
As above, just `btm_asset_id`=ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff
```bash
$./bytomcli sub-control-receiver-tx <account_xprv> <account_id> <btm_asset_id> <spend_amount> <control_program>
```
### Set up a wallet and manage the key

If you have started a bytom node, then you can create an account via `create-key password`, which will generate a `keystore` directory containing the keys under the project directory.

```bash
$ ./bytomcli create-key account_name password   # Create an account named account_name using password
$ ./bytomcli delete-key password pubkey         # Delete account pubkey
$ ./bytomcli reset-password oldpassword newpassword pubkey  # Update password
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
