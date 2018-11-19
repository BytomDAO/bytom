Bytom version 1.0.7 is now available from:

  https://github.com/Bytom/bytom/releases/tag/v1.0.7


Please report bugs using the issue tracker at github:

  https://github.com/Bytom/bytom/issues

How to Upgrade
===============

If you are running an older version, shut it down. Wait until it has quited completely, and then run the new version Bytom.
You can operate according to the user manual.[(Bytom User Manual)](https://bytom.io/wp-content/themes/freddo/images/wallet/BytomUsermanualV1.0_en.pdf)


1.0.7 changelog
================
__Bytom Node__

+ `PR #1409`
    - Support bip44 multi-account hierarchy for deterministic wallets.
+ `PR #1418`
    - The Equity contract arguments support string, integer and boolean as input types.
+ `PR #1430`
    - Add node network performance monitor for list-peers API.
+ `PR #1439`
    - Wallet support recovery from the mnemonic.
+ `PR #1442`
    - Add web socket push notification mechanism for new blocks and new transactions.
+ `PR #1450`
    - API get-block add asset definition for transaction's issue action.
+ `PR #1455`
    - Node support using SOCKS5 connect to Bytom network through a proxy server.
+ `PR #1459`
    - Modify equity compiler to support define/assign/if-else statement, and added the equity compiler tool..
+ `PR #1462`
    - Fixed wrong balance display after deleting the account.
+ `PR #1466`
    - Add update account alias API.
+ `PR #1473`
    - Support Mac using brew to install Bytom.

__Bytom Dashboard__

- Reconstruct the International language framework, change the hard code style into i18n mode.
- Fixed the big number issue.
- Added the mnemonic feature. Create mnemonic page, confirm mnemonic and restore by mnemonic page.
- Updated the Equity contract template.

Credits
--------

Thanks to everyone who directly contributed to this release:

- cancelloveyan
- Colt-Z
- Dkaiju
- HAOYUatHZ
- langyu
- oysheng
- Paladz
- shenao78
- shengling2008
- yahtoo
- zcc0721
- ZhitingLin

And everyone who helped test.
