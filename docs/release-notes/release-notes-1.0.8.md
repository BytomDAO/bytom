Bytom version 1.0.8 is now available from:

  https://github.com/Bytom/bytom/releases/tag/v1.0.8


Please report bugs using the issue tracker at github:

  https://github.com/Bytom/bytom/issues

How to Upgrade
===============

If you are running an older version, shut it down. Wait until it has quited completely, and then run the new version Bytom.
You can operate according to the user manual.[(Bytom User Manual)](https://bytom.io/wp-content/themes/freddo/images/wallet/BytomUsermanualV1.0_en.pdf)


1.0.8 changelog
================
__Bytom Node__

+ `PR #1537`
    - Add mined block subscribe function for easy wallet module subscription.
+ `PR #1539`
    - Discover: add node persistent storage, enabling faster node discovery.
+ `PR #1554`
    - Support Get the seed node from the DNS seed server.
+ `PR #1561`
    - Fix restore wallet will import duplicate key bugs.
+ `PR #1538`
    - Refactor switch code and add test makes the code structure clearer.
+ `PR #1573`
    - Fixed node startup id, preventing the node from regaining the node id every time it starts.
+ `PR #1592`
    - Fix new mined orphan block broadcast bug to prevent invalid blocks from being malicious.
+ `PR #1605`
    - Add no BTM input tx filter, to prevent dust transaction into the transaction pool.
+ `PR #1544`
    - get-raw-block API support return the transaction status
+ `PR #1615`
    - Update the wallet model for support switch chain core database in edge case situation
+ `PR #1585`
    - Strict block header validate rules for preventing irrational block version
+ `PR #1579`
    - limit the max number of orphan blocks which prevent memory attacks that create large numbers of orphan blocks
+ `PR #1582`
    - WebSocket subscriber will receive a raw transaction and corresponding status_fail field when an unconfirmed transaction arrives
+ `PR #1617`
    - Optimize the UTXO manage transaction processing order of block rollback.


__Bytom Dashboard__

- Add the Qr code component for RawTransaction JSON and Signature JSON.

Credits
--------

Thanks to everyone who directly contributed to this release:

- Colt-Z
- HAOYUatHZ
- langyu
- Paladz
- shenao78
- shengling2008
- yahtoo
- zcc0721
- ZhitingLin

And everyone who helped test.
