# equity compiler tool

The equity compiler tool is the equity commandline compiler.

## Requirements

[Go](https://golang.org/doc/install) version 1.8 or higher, with `$GOPATH` set to your preferred directory

Build source code, the build target of the equity compiler commandline tool is `equity`.

```bash
$ make tool
```

then change directory to `equity`, and you can find the tool `equity` :
```bash
$ cd equity
```

## Usage on the commandline

Usage of equity commandline compiler:
```shell
$ ./equity <input_file> [flags]
```

Using help provides you with an explanation of all options.

```shell
$ ./equity --help
```

available flags:
```shell
    --bin        Binary of the contracts in hex.
    --instance   Object of the Instantiated contracts.
    --shift      Function shift of the contracts.
```

## Example

The contents of the contract file `TradeOffer`(without file suffix restrictions) are as follows:
```js
contract TradeOffer(assetRequested: Asset,
                    amountRequested: Amount,
                    seller: Program,
                    cancelKey: PublicKey) locks valueAmount of valueAsset {
  clause trade() {
    lock amountRequested of assetRequested with seller
    unlock valueAmount of valueAsset
  }
  clause cancel(sellerSig: Signature) {
    verify checkTxSig(cancelKey, sellerSig)
    unlock valueAmount of valueAsset
  }
}
```

- Compiler contract file to generate the binary:
```shell
./equity TradeOffer --bin
```

  the return result:
```shell
======= TradeOffer =======
Binary:
547a6413000000007b7b51547ac1631a000000547a547aae7cac
```

- Query the clause shift for contract:
```shell
./equity TradeOffer --shift
```

  the return result:
```shell
======= TradeOffer =======
Clause shift:
    trade:  00000000
    cancel:  13000000
    ending:  1a000000
```

NOTE: 
If the contract contains only one clause, Users don't need clause selector when unlock contract. Furthermore, there is no signification for ending clause shift except for display.

- Instantiated contract with arguments:
```shell
./equity TradeOffer --instance 84fe51a7739e8e2fe28e7042bb114fd6d6abd09cd22af867729ea001c87cd550 1000 0014d6598ab7dce6b04d43f31ad6eed76b18da553e94 7975f3f71ca7f55ecdef53ccf44224d514bc584bc065770bba8dcdb9d7f9ae6c
```

  the return result:
```shell
======= TradeOffer =======
Instantiated program:
207975f3f71ca7f55ecdef53ccf44224d514bc584bc065770bba8dcdb9d7f9ae6c160014d6598ab7dce6b04d43f31ad6eed76b18da553e9402e8032084fe51a7739e8e2fe28e7042bb114fd6d6abd09cd22af867729ea001c87cd550741a547a6413000000007b7b51547ac1631a000000547a547aae7cac00c0
```

When you don't know the order of the contract parameters, you can use the prompt function:
```shell
./equity TradeOffer --instance
```

  the commandline tips:
```shell
======= TradeOffer =======
Instantiated program:
Error: The number of input arguments 0 is less than the number of contract parameters 4
Usage:
  equity TradeOffer <assetRequested> <amountRequested> <seller> <cancelKey>
```

The input contract argument description:

| type | value description |
| ---- | ----------- |
| Boolean | true/1 , false/0 |
| Integer | 0 ~ 2^63-1 |
| Amount | -2^63 ~ 2^63-1 |
| Asset | hex string with length 64 |
| Hash | hex string with length 64 |
| PublicKey | hex string with length 64 |
| Program | hex string |
| String | string with ASCII, e.g., "this is a test string" |
