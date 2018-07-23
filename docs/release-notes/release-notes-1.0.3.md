Bytom version 1.0.3 is now available from:

  https://github.com/Bytom/bytom/releases/tag/v1.0.3


Please report bugs using the issue tracker at github:

  https://github.com/Bytom/bytom/issues

How to Upgrade
===============

If you are running an older version, shut it down. Wait until it has completely, then run the new version Bytom.
You can operate according to the user manual.[(Bytom User Manual)](https://bytom.io/wp-content/themes/freddo/images/wallet/BytomUsermanualV1.0_en.pdf)


1.0.3 changelog
================
Bytom Node
`PR #969`  - Fix x86-32 system exeception on build transaction.
`PR #983`  - API transaction json struct add tx_size field.
`PR #987`  - API Get-block response's transaction struct add mux_id.
`PR #988`  - Add API decode-program.
`PR #1006` - API list-addresses is sort by create time.
`PR #1022` - API list-transactions and get-transaction support return unconfirmed transaction.
`PR #1023` - Add API get-work-json & submit-work-json
`PR #1030` - Add server flag on peer netowork handshake
`PR #1032` - Implementing the UDP Node Discovery Protocol.
`PR #1039` - Modify error model for support high level error message 

Bytom Dashboard
`a51081c`
  - Add progress bar for Sync Status.
  - Modified the frontend for the list unconfirmed Tx.

`3abb9ac`
  - Add Tutorial for first time user.
  - Fixed the filled amount and asset frontend bug.

`f4d6387`
  - Separate the advanced and normal transactions form into two component. Rework the transactions actions.
  - Submit the form when users hit enter.
  - Fixed some react error in new tx pages.
  - When switch pages pop up the warning dialog if the transaction form is filled.

Credits
--------

Thanks to everyone who directly contributed to this release:
- Colt-Z
- freewind
- HAOYUatHZ
- langyu
- oysheng
- Paladz
- shanhuhai5739 
- yahtoo
- ZhitingLin

And everyone who helped test.
