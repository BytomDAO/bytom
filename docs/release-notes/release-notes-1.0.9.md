Bytom version 1.0.9 is now available from:

  https://github.com/Bytom/bytom/releases/tag/v1.0.9


Please report bugs using the issue tracker at github:

  https://github.com/Bytom/bytom/issues

How to Upgrade
===============

If you are running an older version, shut it down. Wait until it has quited completely, and then run the new version Bytom.
You can operate according to the user manual.[(Bytom User Manual)](https://bytom.io/wp-content/themes/freddo/images/wallet/BytomUsermanualV1.0_en.pdf)


1.0.9 changelog
================
__Bytom Node__

+ `PR #1657`
    - Save the index for all history transactions when "txindex" flag is provided for the purpose of future querying.
+ `PR #1659`
    - Add dust transaction filter rule to filer the transaction with dust output amount.
+ `PR #1662`
    - Add a keep_dial option in order to automatically retry connecting to provided peers.
+ `PR #1677`
    - Add a custom node alias feature, support custom the node's name by the configuration.
+ `PR #1687`
    - Support mDNS LAN peer discover to reduce the network bandwidth required for communication.
+ `PR #1692`
    - Add ugly transaction test that may occur in several scenes such as insufficient fee, unbalanced transaction, overflow, and signature fail tests.
+ `PR #1697`
    - Precisely estimate gas for standard transaction and issue transaction.
+ `PR #1698`
    - Add timestamp as random number generator seed number, ensure random number security.


__Bytom Dashboard__

- Update the Json structure and add new form stepper for the create asset page.
- Add the issue asset option under the new transactions page. Support multi-signature under the issue asset transactions.

Credits
--------

Thanks to everyone who directly contributed to this release:

- Agouri
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
