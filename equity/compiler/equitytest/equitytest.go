package equitytest

const TrivialLock = `
contract TrivialLock() locks amount of asset {
  clause trivialUnlock() {
    unlock amount of asset
  }
}
`

const LockWithPublicKey = `
contract LockWithPublicKey(publicKey: PublicKey) locks amount of asset {
  clause unlockWithSig(sig: Signature) {
    verify checkTxSig(publicKey, sig)
    unlock amount of asset
  }
}
`

const LockWithPKHash = `
contract LockWithPublicKeyHash(pubKeyHash: Hash) locks amount of asset {
  clause spend(pubKey: PublicKey, sig: Signature) {
    verify sha3(pubKey) == pubKeyHash
    verify checkTxSig(pubKey, sig)
    unlock amount of asset
  }
}
`

const LockWith2of3Keys = `
contract LockWith3Keys(pubkey1, pubkey2, pubkey3: PublicKey) locks amount of asset {
  clause unlockWith2Sigs(sig1, sig2: Signature) {
    verify checkTxMultiSig([pubkey1, pubkey2, pubkey3], [sig1, sig2])
    unlock amount of asset
  }
}
`

const LockToOutput = `
contract LockToOutput(address: Program) locks amount of asset {
  clause relock() {
    lock amount of asset with address
  }
}
`

const TradeOffer = `
contract TradeOffer(requestedAsset: Asset, requestedAmount: Amount, sellerProgram: Program, sellerKey: PublicKey) locks amount of asset {
  clause trade() {
    lock requestedAmount of requestedAsset with sellerProgram
    unlock amount of asset
  }
  clause cancel(sellerSig: Signature) {
    verify checkTxSig(sellerKey, sellerSig)
    unlock amount of asset
  }
}
`

const EscrowedTransfer = `
contract EscrowedTransfer(agent: PublicKey, sender: Program, recipient: Program) locks amount of asset {
  clause approve(sig: Signature) {
    verify checkTxSig(agent, sig)
    lock amount of asset with recipient
  }
  clause reject(sig: Signature) {
    verify checkTxSig(agent, sig)
    lock amount of asset with sender
  }
}
`

const RevealPreimage = `
contract RevealPreimage(hash: Hash) locks amount of asset {
  clause reveal(string: String) {
    verify sha3(string) == hash
    unlock amount of asset
  }
}
`
const PriceChanger = `
contract PriceChanger(askAmount: Amount, askAsset: Asset, sellerKey: PublicKey, sellerProg: Program) locks valueAmount of valueAsset {
  clause changePrice(newAmount: Amount, newAsset: Asset, sig: Signature) {
    verify checkTxSig(sellerKey, sig)
    lock valueAmount of valueAsset with PriceChanger(newAmount, newAsset, sellerKey, sellerProg)
  }
  clause redeem() {
    lock askAmount of askAsset with sellerProg
    unlock valueAmount of valueAsset
  }
}
`

const TestDefineVar = `
contract TestDefineVar(result: Integer) locks valueAmount of valueAsset {
  clause LockWithMath(left: Integer, right: Integer) {
    define calculate: Integer = left + right
    verify left != calculate
    verify result == calculate
    unlock valueAmount of valueAsset
  }
}
`

const TestAssignVar = `
contract TestAssignVar(result: Integer) locks valueAmount of valueAsset {
  clause LockWithMath(first: Integer, second: Integer) {
    define calculate: Integer = first
    assign calculate = calculate + second
    verify result == calculate
    unlock valueAmount of valueAsset
  }
}
`

const TestSigIf = `
contract TestSigIf(a: Integer, count:Integer) locks valueAmount of valueAsset {
  clause check(b: Integer, c: Integer) {
    verify b != count
    if a > b {
        verify b > c
    } else {
        verify a > c
    }
    unlock valueAmount of valueAsset
  }
}
`
const TestIfAndMultiClause = `
contract TestIfAndMultiClause(a: Integer, cancelKey: PublicKey) locks valueAmount of valueAsset {
  clause check(b: Integer, c: Integer) {
    verify b != c
    if a > b {
        verify a > c
    }
    unlock valueAmount of valueAsset
  }
  clause cancel(sellerSig: Signature) {
    verify checkTxSig(cancelKey, sellerSig)
    unlock valueAmount of valueAsset
  }
}
`

const TestIfNesting = `
contract TestIfNesting(a: Integer, count:Integer) locks valueAmount of valueAsset {
  clause check(b: Integer, c: Integer, d: Integer) {
    verify b != count
    if a > b {
        if d > c {
           verify a > d
        }
        verify d != b
    } else {
        verify a > c
    }
    verify c != count
    unlock valueAmount of valueAsset
  }
  clause cancel(e: Integer, f: Integer) {
    verify a != e
    if a > f {
      verify e > count
    }
    verify f != count
    unlock valueAmount of valueAsset
  }
}
`
const TestConstantMath = `
contract TestConstantMath(result: Integer, hashByte: Hash, hashStr: Hash, outcome: Boolean) locks valueAmount of valueAsset {
  clause calculation(left: Integer, right: Integer, boolResult: Boolean) {
    verify result == left + right + 10
    verify hashByte == sha3(0x31323330)
    verify hashStr == sha3('string')
    verify !outcome
    verify boolResult && (result == left + 20)
    unlock valueAmount of valueAsset
  }
}
`
