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
  * [Create and launch a single node](#create-and-launch-a-single-node)
  * [Issue an asset test](#issue-an-asset-test)
* [Contributing](#contributing)
* [License](#license)

<!-- vim-markdown-toc -->

## What is Bytom?

Bytom is software designed to operate and connect to highly scalable blockchain networks confirming to the Bytom Blockchain Protocol. Each network allows partipicants to define, issue and transfer digitial assets on a multi-asset shared ledger.

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

### Create and launch a single node

When successfully building the project, the `bytom` and `bytomcli` binary should be present in `cmd/bytom/bytom` and `cmd/bytomcli/bytomcli`, respectively. Then, initialize the node:

```bash
$ cd ./cmd/bytom
$ ./bytom init --home ./.bytom
```

After that, you'll see `.bytom` generated in current directory, then launch the single node:

``` bash
$ ./bytom node --home ./.bytom
```

### Issue an asset test

```bash
$ cd ./cmd/bytomcli
$ ./bytomcli issue-test
```

## Contributing

Thank you for considering to help out with the source code! Any contributions are highly appreciated, and we are grateful for even the smallest of fixes!

If you run into an issue, feel free to [file one](https://github.com/Bytom/bytom/issues/) in this repository. We are glad to help!

## License

[AGPL v3](./LICENSE)
