# Use GPG to sign commits and tags

## Install

On MacOS

``` sh
$ brew install pinentry-mac gpg2
```

## Generate key

### Generate key

``` sh
$ /usr/local/Cellar/gnupg/2.2.17/bin/gpg --full-gen-key
```

### Configure 

#### List key

``` sh
$ /usr/local/Cellar/gnupg/2.2.17/bin/gpg --list-secret-keys --keyid-format LONG
/Users/divinerapier/.gnupg/pubring.kbx
-----------------------------------
sec   rsa4096/1111111111111111 2019-08-07 [SC]
      2222222222222222222222221111111111111111
uid                 [ 绝对 ] divinerapier (github.com) <poriter.coco@gmail.com>
ssb   rsa4096/3333333333333333 2019-08-07 [E]
```

#### Export public key

``` sh
$ /usr/local/Cellar/gnupg/2.2.17/bin/gpg --armor --export 1111111111111111
```

Copy it to [github](https://github.com/settings/keys)

#### Configure

``` sh
$ git config --local user.signingKey 1111111111111111
$ git config --local gpg.program /usr/local/Cellar/gnupg/2.2.17/bin/gpg
$ git config --local commit.gpgSign true
```

#### Configure pinentry

``` sh
$ touch  ~/.gnupg/gpg-agent.conf
$ echo "pinentry-program /usr/local/bin/pinentry-mac" >> ~/.gnupg/gpg-agent.conf
$ echo "no-tty" >> ~/.gnupg/gpg.conf
```
