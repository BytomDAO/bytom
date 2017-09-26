Bytom
=====

[![AGPL v3](https://img.shields.io/badge/license-AGPL%20v3-brightgreen.svg)](./LICENSE)

Table of Contents
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

Given the `bytom` node is running, you can use the provoided `issue-test` to test the asset issuance functionality:

```bash
$ cd ./cmd/bytomcli
$ ./bytomcli issue-test
```

### Multiple node

Get the submodule depenency for multi-node test:

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
