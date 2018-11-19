Bytom version 1.0.6 is now available from:

  https://github.com/Bytom/bytom/releases/tag/v1.0.6rc1


Please report bugs using the issue tracker at github:

  https://github.com/Bytom/bytom/issues

How to Upgrade
===============

If you are running an older version, shut it down. Wait until it has quited completely, and then run the new version Bytom.
You can operate according to the user manual.[(Bytom User Manual)](https://bytom.io/wp-content/themes/freddo/images/wallet/BytomUsermanualV1.0_en.pdf)


1.0.6 changelog
================
__Bytom Node__

+ `PR #1316`
    - The default path of user data in the Mac environment is changed to ~/Library/Application Support/Bytom.
+ `PR #1324`
    - Fix bug for can't process new block due to system unexpectedly crashes.
+ `PR #1323`
    - Support using the mnemonic code for generating deterministic keys(BIP39).
+ `PR #1336`
    - Fix bug for concurrent map access in p2p status broadcast.
+ `PR #1338`
    - Fix API restore-wallet can't restore asset-alias error.
+ `PR #1343`
    - Add witness argument to transaction-related API response.
+ `PR #1369`
    - upgrade the estimate gas function to handle the edge case.
+ `PR #1368`
    - List-balances and list-unspent-outputs API support filter by account ID or account alias.
+ `PR #1365`
    - Add build-chain-transactions API to support intelligent merged BTM UTXOs and sent out the chain transactions.
+ `PR #1378`
    - Add submit-block API to support raw block submission to the remote node.


__Bytom Dashboard__

- Relayout the transaction item display.
- Simplified the transaction details page.
- Add the confirmation page for the normal transaction page.
- Relayout the normal transaction page.
- Add the multiple addresses functions in the normal transaction page.

Credits
--------

Thanks to everyone who directly contributed to this release:

- Colt-Z
- HAOYUatHZ
- huwenchao
- langyu
- oysheng
- Paladz
- shenao78
- yahtoo
- zcc0721
- ZhitingLin

And everyone who helped test.
