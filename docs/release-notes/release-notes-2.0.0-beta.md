Bytom version 1.1.1 is now available from:

  https://github.com/Bytom/bytom/releases/tag/v1.1.1


Please report bugs using the issue tracker at github:

  https://github.com/Bytom/bytom/issues

How to Upgrade
===============

Please notice new version bytom path is $GOPATH/src/github.com/bytom/bytom if you install bytom from source code.  
If you are running an older version, shut it down. Wait until it has quited completely, and then run the new version Bytom.
You can operate according to the user manual.[(Bytom User Manual)](https://bytom.io/wp-content/themes/freddo/images/wallet/BytomUsermanualV1.0_en.pdf)


1.1.0 changelog
================
__Bytom Node__

+ [`PR #1851`](https://github.com/Bytom/bytom/pull/1852/files)
    - Add the new fork choice rule, which follow the chain containing the justified checkpoint of the greatest height.
+ [`PR #1855`](https://github.com/Bytom/bytom/pull/1855/files)
    - According to the number of validator mortgage and the number of votes received, a maximum of 10 validators are finally selected.
+ [`PR #1858`](https://github.com/Bytom/bytom/pull/1858/files) 
    - Add the structure of verification message, and auth whether a verification message is legal. 
+ [`PR #1868`](https://github.com/Bytom/bytom/pull/1868/files) 
    - When a new block comes, consensus algorithm will update the information in the checkpoint and make it persistent.
+ [`PR #1935`](https://github.com/Bytom/bytom/pull/1868/files) 
    - In the new consensus algorithm, nodes take turns to generate blocks every 6 seconds.

Credits
--------

Thanks to everyone who directly contributed to this release:

- shenao78
- DeKaiju
- iczc
- Paladz
- zcc0721
- ZhitingLin

And everyone who helped test.
