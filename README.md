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

### Initialize

First of all, initialize the node:

```bash
$ cd ./cmd/bytomd
$ ./bytomd init --chain_id testnet
```

There are two options for the flag `--chain_id`:

- `testnet`: connect to the testnet.
- `mainnet`: standalone mode.

After that, you'll see `.bytomd` generated in current directory, then launch the node.

### launch

``` bash
$ ./bytomd node --mining
```

available flags for `bytomd node`:

```
      --auth.disable                Disable rpc access authenticate
      --mining                      Enable mining
      --p2p.dial_timeout int        Set dial timeout (default 3)
      --p2p.handshake_timeout int   Set handshake timeout (default 30)
      --p2p.laddr string            Node listen address.
      --p2p.max_num_peers int       Set max num peers (default 50)
      --p2p.pex                     Enable Peer-Exchange
      --p2p.seeds string            Comma delimited host:port seed nodes
      --p2p.skip_upnp               Skip UPNP configuration
      --prof_laddr string           Use http to profile bytomd programs
      --wallet.disable              Disable wallet
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

Every asset-related action is trigger via sending a transaction, which requires two steps to complete, i.e., `build-transaction` and `sign-submit-transaction`. Don't forget to enable `--mining`.

#### Issue

##### `build-transaction`

Let's say, issue 10000 gold to alice:

```bash
$ ./bytomcli build-transaction -t issue alice gold 10000 --alias
Template Type: issue
{"allow_additional_actions":false,"local":true,"raw_transaction":"07010002019c01019901fb77ab11fafc5cf0446d552724aed272b71cfc7ac7d032d2b1de8890e482d0beffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80c0b1ca9412000130cd016ea069766baa2074615605ef8fd37176bb5a75ec3d51d080a2e3a5441e84e255a339ea9ee117805151ad696c00c0a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a000100012c00088fccd0ad39658582ab9df804775507462b17cdcdf101feb2ab4fc049092e8557a3a70a67cc2f0d60904e004f64eaa1cae4cf14f775251c5eb786376d2cf218e488dcea79826b9ce56d0a000000012b766baa20279bdb8c0925d3bbf27f3262e40567d176ade2a64ee0201217651f278a4133405151ad696c00c000020153ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80e6ecc09412012b766baa2017a127adf137d75730fa7098028204ea04e20a6ceb415961a6d0def86bc1015d5151ad696c00c00000014fab9df804775507462b17cdcdf101feb2ab4fc049092e8557a3a70a67cc2f0d60904e012b766baa201f7c723946d2a49af4e069b8df25c98d347d9e720e440a32131c07f7aecf930a5151ad696c00c0000000","signing_instructions":[{"position":0,"witness_components":[{"keys":[{"derivation_path":["010100000000000000","4d00000000000000"],"xpub":"f6f2a78fa8242bf09d073af8c0272272d4b53b3f8469ec2906fe9eae513bd7236653f4227aec2bafc6a301a35dda53e824ae7d4f7db25f83d4666d6e3e30628f"}],"quorum":1,"signatures":null,"type":"signature"}]},{"position":1,"witness_components":[{"keys":[{"derivation_path":["000300000000000000"],"xpub":"f6f2a78fa8242bf09d073af8c0272272d4b53b3f8469ec2906fe9eae513bd7236653f4227aec2bafc6a301a35dda53e824ae7d4f7db25f83d4666d6e3e30628f"}],"quorum":1,"signatures":null,"type":"signature"}]}]}
```

The response of `build-transaction` will be used in the following `sign-submit-transaction` command.

##### `sign-submit-transaction`

```bash
$ ./bytomcli sign-submit-transaction '{"allow_additional_actions":false,"local":true,"raw_transaction":"07010002019c01019901fb77ab11fafc5cf0446d552724aed272b71cfc7ac7d032d2b1de8890e482d0beffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80c0b1ca9412000130cd016ea069766baa2074615605ef8fd37176bb5a75ec3d51d080a2e3a5441e84e255a339ea9ee117805151ad696c00c0a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a000100012c00088fccd0ad39658582ab9df804775507462b17cdcdf101feb2ab4fc049092e8557a3a70a67cc2f0d60904e004f64eaa1cae4cf14f775251c5eb786376d2cf218e488dcea79826b9ce56d0a000000012b766baa20279bdb8c0925d3bbf27f3262e40567d176ade2a64ee0201217651f278a4133405151ad696c00c000020153ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80e6ecc09412012b766baa2017a127adf137d75730fa7098028204ea04e20a6ceb415961a6d0def86bc1015d5151ad696c00c00000014fab9df804775507462b17cdcdf101feb2ab4fc049092e8557a3a70a67cc2f0d60904e012b766baa201f7c723946d2a49af4e069b8df25c98d347d9e720e440a32131c07f7aecf930a5151ad696c00c0000000","signing_instructions":[{"position":0,"witness_components":[{"keys":[{"derivation_path":["010100000000000000","4d00000000000000"],"xpub":"f6f2a78fa8242bf09d073af8c0272272d4b53b3f8469ec2906fe9eae513bd7236653f4227aec2bafc6a301a35dda53e824ae7d4f7db25f83d4666d6e3e30628f"}],"quorum":1,"signatures":null,"type":"signature"}]},{"position":1,"witness_components":[{"keys":[{"derivation_path":["000300000000000000"],"xpub":"f6f2a78fa8242bf09d073af8c0272272d4b53b3f8469ec2906fe9eae513bd7236653f4227aec2bafc6a301a35dda53e824ae7d4f7db25f83d4666d6e3e30628f"}],"quorum":1,"signatures":null,"type":"signature"}]}]}'
{
  "txid": "69eb0c8f7622c350e0b253a55bfbdc2ec06a396ca488ea34a104696c6cf9adda"
}
```

When the transaction is on-chain, query the balances:

```bash
# alice should have 10000 gold now
$ ./bytomcli list-balances
0 :
{
  "account_alias": "alice",
  "account_id": "08FU3M22G0A02",
  "amount": 10000,
  "asset_alias": "gold",
  "asset_id": "ab9df804775507462b17cdcdf101feb2ab4fc049092e8557a3a70a67cc2f0d60"
}
1 :
{
  "account_alias": "alice",
  "account_id": "08FU3M22G0A02",
  "amount": 135408000000000,
  "asset_alias": "btm",
  "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
}
```

#### Spend

Alice pays Bob `<payment_amount>`, e.g., 1000, `gold`:

##### `create-account-receiver`

Before you transfer an asset to another account, you have to know his `control_program`. This means the receiver needs to send you his `control_program` first.

```bash
$ ./bytomcli create-account-receiver bob
{
  "control_program": "766baa20510e0df72e4e01363fdb07d57b8417a19c6517aad8cd6129f495c52188c246f25151ad696c00c0",
  "expires_at": "2018-02-01T17:55:10.895099+08:00"
}
```

##### `build-transaction`

```bash
# ./bytomcli build-transaction -t spend <sender_account> <asset> <amount> --alias -r <receiver_control_program>
$ ./bytomcli build-transaction -t spend alice gold 1000 --alias -r 766baa2012ba2491f96011afe9ad1ad6423887b1f2b904e5e8185cdb47d14fb2f7aa19615151ad696c00c0
Template Type: spend
{"allow_additional_actions":false,"local":true,"raw_transaction":"07010002019d01019a01fb77ab11fafc5cf0446d552724aed272b71cfc7ac7d032d2b1de8890e482d0beffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80c0b1ca9412000131cd029301a069766baa2033400ee7b1c93955b4c6d9532141e1bca263c3f546f409b1b75984f2fe8a1ba05151ad696c00c0a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a0001000193010190011fd6fe88a17024f66671e32147fb42c6f8a8f48e7c3af1f7ff79df4d80388bf1ab9df804775507462b17cdcdf101feb2ab4fc049092e8557a3a70a67cc2f0d60904e01012b766baa201f7c723946d2a49af4e069b8df25c98d347d9e720e440a32131c07f7aecf930a5151ad696c00c0a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a000100030153ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80e6ecc09412012b766baa2018c27de38474915cd491bedd21625506629d1d09437536932f697d62663034e05151ad696c00c00000014fab9df804775507462b17cdcdf101feb2ab4fc049092e8557a3a70a67cc2f0d60a846012b766baa202b39e0425fab029c4dfa48dea8e17aa0a89eae22c1fe357d015b233d5facd0e05151ad696c00c00000014fab9df804775507462b17cdcdf101feb2ab4fc049092e8557a3a70a67cc2f0d60e807012b766baa2012ba2491f96011afe9ad1ad6423887b1f2b904e5e8185cdb47d14fb2f7aa19615151ad696c00c0000000","signing_instructions":[{"position":0,"witness_components":[{"keys":[{"derivation_path":["010100000000000000","7501000000000000"],"xpub":"f6f2a78fa8242bf09d073af8c0272272d4b53b3f8469ec2906fe9eae513bd7236653f4227aec2bafc6a301a35dda53e824ae7d4f7db25f83d4666d6e3e30628f"}],"quorum":1,"signatures":null,"type":"signature"}]},{"position":1,"witness_components":[{"keys":[{"derivation_path":["010100000000000000","6500000000000000"],"xpub":"f6f2a78fa8242bf09d073af8c0272272d4b53b3f8469ec2906fe9eae513bd7236653f4227aec2bafc6a301a35dda53e824ae7d4f7db25f83d4666d6e3e30628f"}],"quorum":1,"signatures":null,"type":"signature"}]}]}
```

##### `sign-submit-transaction`

```bash
$ ./bytomcli sign-submit-transaction '{"allow_additional_actions":false,"local":true,"raw_transaction":"07010002019d01019a01fb77ab11fafc5cf0446d552724aed272b71cfc7ac7d032d2b1de8890e482d0beffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80c0b1ca9412000131cd029301a069766baa2033400ee7b1c93955b4c6d9532141e1bca263c3f546f409b1b75984f2fe8a1ba05151ad696c00c0a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a0001000193010190011fd6fe88a17024f66671e32147fb42c6f8a8f48e7c3af1f7ff79df4d80388bf1ab9df804775507462b17cdcdf101feb2ab4fc049092e8557a3a70a67cc2f0d60904e01012b766baa201f7c723946d2a49af4e069b8df25c98d347d9e720e440a32131c07f7aecf930a5151ad696c00c0a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a000100030153ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80e6ecc09412012b766baa2018c27de38474915cd491bedd21625506629d1d09437536932f697d62663034e05151ad696c00c00000014fab9df804775507462b17cdcdf101feb2ab4fc049092e8557a3a70a67cc2f0d60a846012b766baa202b39e0425fab029c4dfa48dea8e17aa0a89eae22c1fe357d015b233d5facd0e05151ad696c00c00000014fab9df804775507462b17cdcdf101feb2ab4fc049092e8557a3a70a67cc2f0d60e807012b766baa2012ba2491f96011afe9ad1ad6423887b1f2b904e5e8185cdb47d14fb2f7aa19615151ad696c00c0000000","signing_instructions":[{"position":0,"witness_components":[{"keys":[{"derivation_path":["010100000000000000","7501000000000000"],"xpub":"f6f2a78fa8242bf09d073af8c0272272d4b53b3f8469ec2906fe9eae513bd7236653f4227aec2bafc6a301a35dda53e824ae7d4f7db25f83d4666d6e3e30628f"}],"quorum":1,"signatures":null,"type":"signature"}]},{"position":1,"witness_components":[{"keys":[{"derivation_path":["010100000000000000","6500000000000000"],"xpub":"f6f2a78fa8242bf09d073af8c0272272d4b53b3f8469ec2906fe9eae513bd7236653f4227aec2bafc6a301a35dda53e824ae7d4f7db25f83d4666d6e3e30628f"}],"quorum":1,"signatures":null,"type":"signature"}]}]}'


{
  "txid": "a07a2289682d66de2affd7d2aa5bb4420171a2c4edb91d6bada3021af2214872"
}
```

```bash
$./bytomcli list-balances
0 :
{
  "account_alias": "alice",
  "account_id": "08FU3M22G0A02",
  "amount": 9000,
  "asset_alias": "gold",
  "asset_id": "ab9df804775507462b17cdcdf101feb2ab4fc049092e8557a3a70a67cc2f0d60"
}
1 :
{
  "account_alias": "alice",
  "account_id": "08FU3M22G0A02",
  "amount": 310752000000000,
  "asset_alias": "btm",
  "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
}
2 :
{
  "account_alias": "bob",
  "account_id": "08FU3U04G0A04",
  "amount": 1000,
  "asset_alias": "gold",
  "asset_id": "ab9df804775507462b17cdcdf101feb2ab4fc049092e8557a3a70a67cc2f0d60"
}
```

#### Transfer BTM

```bash
# ./bytomcli build-transaction -t spend <sender_account> <asset> <amount> --alias -r <receiver_control_program>
$ ./bytomcli build-transaction -t spend alice btm 100000000000 --alias -r 766baa2012ba2491f96011afe9ad1ad6423887b1f2b904e5e8185cdb47d14fb2f7aa19615151ad696c00c0
Template Type: spend
{"allow_additional_actions":false,"local":true,"raw_transaction":"07010002019d01019a01fb77ab11fafc5cf0446d552724aed272b71cfc7ac7d032d2b1de8890e482d0beffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80c0b1ca9412000131cd02d601a069766baa20b06f9f698226dd2ba4e6d9db2d32db68ce3c9b2a35ad5b5570d2159e01d394af5151ad696c00c0a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a000100019d01019a01fb77ab11fafc5cf0446d552724aed272b71cfc7ac7d032d2b1de8890e482d0beffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80c0b1ca9412000131cd024801a069766baa2068ea529208b33a5294ee96f7ee0ce43609addd7a89f1b4c0ed1ded25841948d15151ad696c00c0a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a000100030153ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80e6ecc09412012b766baa20c37b4dfcabed2ef74fb05122f10729629c15b71914d151730f933e5a4d4a271d5151ad696c00c000000153ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80f0d586a00f012b766baa206e58e27b81bef8c5c993491859ee35572515ad048db8147146ed590ac95c1daa5151ad696c00c000000153ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80d0dbc3f402012b766baa2012ba2491f96011afe9ad1ad6423887b1f2b904e5e8185cdb47d14fb2f7aa19615151ad696c00c0000000","signing_instructions":[{"position":0,"witness_components":[{"keys":[{"derivation_path":["010100000000000000","ba01000000000000"],"xpub":"f6f2a78fa8242bf09d073af8c0272272d4b53b3f8469ec2906fe9eae513bd7236653f4227aec2bafc6a301a35dda53e824ae7d4f7db25f83d4666d6e3e30628f"}],"quorum":1,"signatures":null,"type":"signature"}]},{"position":1,"witness_components":[{"keys":[{"derivation_path":["010100000000000000","2901000000000000"],"xpub":"f6f2a78fa8242bf09d073af8c0272272d4b53b3f8469ec2906fe9eae513bd7236653f4227aec2bafc6a301a35dda53e824ae7d4f7db25f83d4666d6e3e30628f"}],"quorum":1,"signatures":null,"type":"signature"}]}]}
```

```bash
$ ./bytomcli sign-submit-transaction '{"allow_additional_actions":false,"local":true,"raw_transaction":"07010002019d01019a01fb77ab11fafc5cf0446d552724aed272b71cfc7ac7d032d2b1de8890e482d0beffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80c0b1ca9412000131cd02d601a069766baa20b06f9f698226dd2ba4e6d9db2d32db68ce3c9b2a35ad5b5570d2159e01d394af5151ad696c00c0a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a000100019d01019a01fb77ab11fafc5cf0446d552724aed272b71cfc7ac7d032d2b1de8890e482d0beffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80c0b1ca9412000131cd024801a069766baa2068ea529208b33a5294ee96f7ee0ce43609addd7a89f1b4c0ed1ded25841948d15151ad696c00c0a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a000100030153ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80e6ecc09412012b766baa20c37b4dfcabed2ef74fb05122f10729629c15b71914d151730f933e5a4d4a271d5151ad696c00c000000153ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80f0d586a00f012b766baa206e58e27b81bef8c5c993491859ee35572515ad048db8147146ed590ac95c1daa5151ad696c00c000000153ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80d0dbc3f402012b766baa2012ba2491f96011afe9ad1ad6423887b1f2b904e5e8185cdb47d14fb2f7aa19615151ad696c00c0000000","signing_instructions":[{"position":0,"witness_components":[{"keys":[{"derivation_path":["010100000000000000","ba01000000000000"],"xpub":"f6f2a78fa8242bf09d073af8c0272272d4b53b3f8469ec2906fe9eae513bd7236653f4227aec2bafc6a301a35dda53e824ae7d4f7db25f83d4666d6e3e30628f"}],"quorum":1,"signatures":null,"type":"signature"}]},{"position":1,"witness_components":[{"keys":[{"derivation_path":["010100000000000000","2901000000000000"],"xpub":"f6f2a78fa8242bf09d073af8c0272272d4b53b3f8469ec2906fe9eae513bd7236653f4227aec2bafc6a301a35dda53e824ae7d4f7db25f83d4666d6e3e30628f"}],"quorum":1,"signatures":null,"type":"signature"}]}]}'


{
  "txid": "91a265de33ffd709bb7135bd6c7a0e4e487e771d553a07f0b5513185ee79d3e0"
}
```

```bash
$ ./bytomcli list-balances
0 :
{
  "account_alias": "alice",
  "account_id": "08FU3M22G0A02",
  "amount": 9000,
  "asset_alias": "gold",
  "asset_id": "ab9df804775507462b17cdcdf101feb2ab4fc049092e8557a3a70a67cc2f0d60"
}
1 :
{
  "account_alias": "alice",
  "account_id": "08FU3M22G0A02",
  "amount": 530924000000000,
  "asset_alias": "btm",
  "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
}
2 :
{
  "account_alias": "bob",
  "account_id": "08FU3U04G0A04",
  "amount": 1000,
  "asset_alias": "gold",
  "asset_id": "ab9df804775507462b17cdcdf101feb2ab4fc049092e8557a3a70a67cc2f0d60"
}
3 :
{
  "account_alias": "bob",
  "account_id": "08FU3U04G0A04",
  "amount": 100000000000,
  "asset_alias": "btm",
  "asset_id": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
}
```

## Running Bytom in Docker

Ensure your [Docker](https://www.docker.com/) version is 17.05 or higher.

```bash
$ docker build -t bytom .
```

## Contributing

Thank you for considering helping out with the source code! Any contributions are highly appreciated, and we are grateful for even the smallest of fixes!

If you run into an issue, feel free to [file one](https://github.com/Bytom/bytom/issues/) in this repository. We are glad to help!

## License

[AGPL v3](./LICENSE)
