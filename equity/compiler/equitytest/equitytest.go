package equitytest

const TrivialLock = `
contract TrivialLock() locks locked {
  clause trivialUnlock() {
    unlock locked
  }
}
`

const LockWithPublicKey = `
contract LockWithPublicKey(publicKey: PublicKey) locks locked {
  clause unlockWithSig(sig: Signature) {
    verify checkTxSig(publicKey, sig)
    unlock locked
  }
}
`

const LockWithPKHash = `
contract LockWithPublicKeyHash(pubKeyHash: Hash) locks value {
  clause spend(pubKey: PublicKey, sig: Signature) {
    verify sha3(pubKey) == pubKeyHash
    verify checkTxSig(pubKey, sig)
    unlock value
  }
}
`

const LockWith2of3Keys = `
contract LockWith3Keys(pubkey1, pubkey2, pubkey3: PublicKey) locks locked {
  clause unlockWith2Sigs(sig1, sig2: Signature) {
    verify checkTxMultiSig([pubkey1, pubkey2, pubkey3], [sig1, sig2])
    unlock locked
  }
}
`

const LockToOutput = `
contract LockToOutput(address: Program) locks locked {
  clause relock() {
    lock locked with address
  }
}
`

const TradeOffer = `
contract TradeOffer(requestedAsset: Asset, requestedAmount: Amount, sellerProgram: Program, sellerKey: PublicKey) locks offered {
  clause trade() requires payment: requestedAmount of requestedAsset {
    lock payment with sellerProgram
    unlock offered
  }
  clause cancel(sellerSig: Signature) {
    verify checkTxSig(sellerKey, sellerSig)
    lock offered with sellerProgram
  }
}
`

const EscrowedTransfer = `
contract EscrowedTransfer(agent: PublicKey, sender: Program, recipient: Program) locks value {
  clause approve(sig: Signature) {
    verify checkTxSig(agent, sig)
    lock value with recipient
  }
  clause reject(sig: Signature) {
    verify checkTxSig(agent, sig)
    lock value with sender
  }
}
`

const CollateralizedLoan = `
contract CollateralizedLoan(balanceAsset: Asset, balanceAmount: Amount, finalHeight: Integer, lender: Program, borrower: Program) locks collateral {
  clause repay() requires payment: balanceAmount of balanceAsset {
    lock payment with lender
    lock collateral with borrower
  }
  clause default() {
    verify above(finalHeight)
    lock collateral with lender
  }
}
`

const RevealPreimage = `
contract RevealPreimage(hash: Hash) locks value {
  clause reveal(string: String) {
    verify sha3(string) == hash
    unlock value
  }
}
`

const PriceChanger = `
contract PriceChanger(askAmount: Amount, askAsset: Asset, sellerKey: PublicKey, sellerProg: Program) locks offered {
  clause changePrice(newAmount: Amount, newAsset: Asset, sig: Signature) {
    verify checkTxSig(sellerKey, sig)
    lock offered with PriceChanger(newAmount, newAsset, sellerKey, sellerProg)
  }
  clause redeem() requires payment: askAmount of askAsset {
    lock payment with sellerProg
    unlock offered
  }
}
`

const CallOptionWithSettlement = `
contract CallOptionWithSettlement(strikePrice: Amount,
                    strikeCurrency: Asset,
                    sellerProgram: Program,
                    sellerKey: PublicKey,
                    buyerKey: PublicKey,
                    finalHeight: Integer) locks underlying {
  clause exercise(buyerSig: Signature)
                 requires payment: strikePrice of strikeCurrency {
    verify below(finalHeight)
    verify checkTxSig(buyerKey, buyerSig)
    lock payment with sellerProgram
    unlock underlying
  }
  clause expire() {
    verify above(finalHeight)
    lock underlying with sellerProgram
  }
  clause settle(sellerSig: Signature, buyerSig: Signature) {
    verify checkTxSig(sellerKey, sellerSig)
    verify checkTxSig(buyerKey, buyerSig)
    unlock underlying
  }
}
`

const OneTwo = `
contract Two(b, c: Program, expirationHeight: Integer) locks value {
  clause redeem() {
    verify below(expirationHeight)
    lock value with b
  }
  clause default() {
    verify above(expirationHeight)
    lock value with c
  }
}
contract One(a, b, c: Program, switchHeight, blockHeight: Integer) locks value {
  clause redeem() {
    verify below(switchHeight)
    lock value with a
  }
  clause switch() {
    verify above(switchHeight)
    lock value with Two(b, c, blockHeight)
  }
}
`
