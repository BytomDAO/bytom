Bytom version 1.0.5 is now available from:

  https://github.com/Bytom/bytom/releases/tag/v1.0.5


Please report bugs using the issue tracker at github:

  https://github.com/Bytom/bytom/issues

How to Upgrade
===============

If you are running an older version, shut it down. Wait until it has quited completely, and then run the new version Bytom.
You can operate according to the user manual.[(Bytom User Manual)](https://bytom.io/wp-content/themes/freddo/images/wallet/BytomUsermanualV1.0_en.pdf)


1.0.5 changelog
================
__Bytom Node__

+ `PR #1196`
    - Remove the old p2p TCP peer exchange module.
+ `PR #1204`
    - Add paging to the three APIs: list-unspent-outputs,list-transactions, and list-addresses.
+ `PR #1208`
    - Add sync completion status broadcasting.
+ `PR #1218`
    - Fix the Unmarshal bug at getting an external asset.
+ `PR #1233`
    - Add check-key-password API.
+ `PR #1219`
    - Add get-coinbase-arbitrary, set-coinbase-arbitrary APIs.
+ `PR #1232`
    - Fix multi-sign process only being signed once if signed by the same password.
+ `PR #1228`
    - Upgrade txpool to prevent dropping orphan transaction.
+ `PR #1241`
    - Add function to delete expired orphan block.
+ `PR #1253`
    - Add support to filter by alias for list-accounts API.
+ `PR #1245`
    - API create-asset supports user custom smart contract.
+ `PR #1258`
    - Add node upgrade notification mechanism.
+ `PR #1264`
    - Improve the mining pool new block updating timing.
+ `PR #1262`
    - Add spv support for full node，which mainly includes filtering address and sending Merkle block.

__Bytom Dashboard__

- Add Chinese translation to equity contract.
- Fix password will be frozen for 5 mins when the password is wrong.
- Add transaction details before signing an advanced transaction.
- Add version tag and version update notification.

__Equity Contract frontend__

- Add contract template of RevealPreimage support for entering any character
- Asset selection box support BTM，expect when locking value

Credits
--------

Thanks to everyone who directly contributed to this release:

- Colt-Z
- HAOYUatHZ
- langyu
- oysheng
- Paladz
- shanhuhai5739
- shenao78
- successli
- yahtoo
- zcc0721
- ZhitingLin

And everyone who helped test.
