Bytom version 1.1.0 is now available from:

  https://github.com/Bytom/bytom/releases/tag/v1.1.0


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

+ [`PR #1805`](https://github.com/Bytom/bytom/pull/1805)
    - Correct bytom go import path to github.com/bytom/bytom. Developer can use go module to manage dependency of bytom. 
+ [`PR #1815`](https://github.com/Bytom/bytom/pull/1815) 
    - Add asynchronous validate transactions function to optimize the performance of validating and saving block. 

__Bytom Dashboard__

+ [`PR #1829`](https://github.com/Bytom/bytom/pull/1829) 
    - Fixed the decimals type string to integer in create asset page.

Credits
--------

Thanks to everyone who directly contributed to this release:

- DeKaiju
- iczc
- Paladz
- zcc0721
- ZhitingLin

And everyone who helped test.
