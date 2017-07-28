# merkleeyes

[![CircleCI](https://circleci.com/gh/tendermint/merkleeyes.svg?style=svg)](https://circleci.com/gh/tendermint/merkleeyes)

A simple [ABCI application](http://github.com/tendermint/abci) serving a [merkle-tree key-value store](http://github.com/tendermint/merkleeyes/iavl) 

# Use

Merkleeyes allows inserts and removes by key, and queries by key or index.
Inserts and removes happen through the `DeliverTx` message, while queries happen through the `Query` message.
`CheckTx` simply mirrors `DeliverTx`.

# Formatting

A transaction is a serialized request on the key-value store, using [go-wire](https://github.com/tendermint/go-wire)
for serialization.

Each function (set/insert, remove, get-by-key, get-by-index) has a corresponding type byte:

```
DeliverTx/CheckTx
--------
- 0x01 for a set
- 0x02 for a remove

Query
--------
- 0x01 for 'by Key'
- 0x02 for 'by Index'
```

The format of a transaction is:

```
<TypeByte> <Encode(key)> <Encode(value)>
```

which translates to (where `Encode()` is the `go-wire` encoding function):

```
ByteType ByteVarintSizeKey BytesVarintKey BytesKey ByteVarintSizeValue BytesVarintValue BytesValue
```

For instance, to insert the key-value pair `(eric, clapton)`, you would submit the following bytes in an DeliverTx:

```
010104657269630107636c6170746f6e
```

Here's a session from the [abci-cli](https://tendermint.com/intro/getting-started/first-abci):

```
> append_tx 0x010104657269630107636c6170746f6e
-> code: OK

> query 0x01010465726963                  
-> code: OK
-> data: {clapton}
```

# Poem

```
writing down, my checksum
waiting for the, data to come
no need to pray for integrity
thats cuz I use, a merkle tree

grab the root, with a quick hash run
if the hash works out,
it must have been done

theres no need, for trust to arise
thanks to the crypto
now that I can merkleyes

take that data, merklize
ye, I merklize ...

then the truth, begins to shine
the inverse of a hash, you will never find
and as I watch, the dataset grow
producing a proof, is never slow

Where do I find, the will to hash
How do I teach it?
It doesn't pay in cash
Bitcoin, here, I've realized
Thats what I need now,
cuz real currencies merklize
-EB
```
