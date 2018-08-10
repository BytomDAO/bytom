Bytom version 1.0.4 is now available from:

  https://github.com/Bytom/bytom/releases/tag/v1.0.4


Please report bugs using the issue tracker at github:

  https://github.com/Bytom/bytom/issues

How to Upgrade
===============

If you are running an older version, shut it down. Wait until it has completely, then run the new version Bytom.
You can operate according to the user manual.[(Bytom User Manual)](https://bytom.io/wp-content/themes/freddo/images/wallet/BytomUsermanualV1.0_en.pdf)


1.0.4 changelog
================
__Bytom Node__

+ `PR #1104`
    - Add block fast sync function.
+ `PR #1048`
    - Sort actions by original list for function MergeSpendAction.
+ `PR #1081`
    - Add API list-pubkeys.
+ `PR #1098`
    - Add API wallet-info to acquire rescanning wallet schedule.
+ `PR #1112`
    - Wallet support spends unconfirmed utxo.
+ `PR #1115`
    - Add bytomd command line parameter `--log_level` to set log level.
+ `PR #1118`
    - Add network access control api, include list-peers,connect-peer,disconnect-peer.
+ `PR #1124`
    - Fix a security bug that might attack Bytom server.
+ `PR #1126`  
    - Optimize the gas estimation for the multi-signed transaction.
+ `PR #1130`  
    - Add tx_id and input_id to the decode-raw-transaction API response.
+ `PR #1133`
    - Reorganize error codes and messages
+ `PR #1139`
    - Fix p2p node discover table delete bug
+ `PR #1141`
    - Delete unconfirmed transaction from the dashboard if it has been double spend 
+ `PR #1142`
    - Add simd support for tensority, including compilation option and command line flag (`--simd.enable`).
+ `PR #1149`
    - Optimize wallet utxo select algorithm on build transaction.

__Bytom Dashboard__

+ `PR #1143`
    - Update the password field to prevent browser remember password.
    - Add the rescan Wallet button for the balances page.
    - Restyled the backup & restore pages.
    - Add the terminal pop up modal in the setting page.
`PR #1169`
    - updated error message display in submitted form.

__Equity Contract frontend__

+ `PR #1144`
    - Add 8 contracts to the lock page.
    - Render the static component in the unlock page.
    - Setup and configured the equity project into production environment.
    - Unlock page get data from contract program for dynamic rendering.
    - Build actions based on different contract templates.

Credits
--------

Thanks to everyone who directly contributed to this release:
- Broadroad
- Colt-Z
- HAOYUatHZ
- langyu
- oysheng
- Paladz
- RockerFlower
- shanhuhai5739
- shenao78
- successli
- yahtoo
- zcc0721
- ZhitingLin

And everyone who helped test.
