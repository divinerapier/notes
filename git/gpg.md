# Use GPG to sign commits and tags

## Install

### On MacOS

``` sh
$ brew install pinentry-mac gpg2
$ touch  ~/.gnupg/gpg-agent.conf
$ echo "pinentry-program /usr/local/bin/pinentry-mac" >> ~/.gnupg/gpg-agent.conf
```

### On Ubuntu

``` bash
$ sudo apt install gunpg2
```

## Generate key

### Generate key

macos

``` sh
$ gpg --full-gen-key
```

ubuntu

``` bash
$ gpg2 --full-gen-key
```

### Configure 

#### List key

``` sh
$ gpg --list-secret-keys --keyid-format LONG
~/.gnupg/pubring.kbx
-----------------------------------
sec   rsa4096/1111111111111111 2019-08-07 [SC]
      2222222222222222222222221111111111111111
uid                 [ 绝对 ] divinerapier (github.com) <poriter.coco@gmail.com>
ssb   rsa4096/3333333333333333 2019-08-07 [E]
```

#### Export public key

``` sh
$ gpg --armor --export 1111111111111111
```

Copy it to [github](https://github.com/settings/keys)

#### Configure

``` sh
$ git config --local user.signingKey 1111111111111111
$ git config --local gpg.program $(which gpg)
$ git config --local commit.gpgSign true
```

#### Configure (Optional)

``` sh
$ echo "no-tty" >> ~/.gnupg/gpg.conf
```

## FAQ

1. gpg: signing failed: Inappropriate ioctl for device

``` bash
$ echo "export GPG_TTY=$(tty)" >> ~/.bashrc
```

2. GPG Hangs When Private Keys are Accessed

``` bash
# restart gpg-agent: refers to 4, 5
$ gpgconf --kill gpg-agent
```

## Reference

1. https://help.github.com/en/articles/telling-git-about-your-signing-key   
2. https://ducfilan.wordpress.com/2017/03/10/the-git-error-gpg-failed-to-sign-the-data/
3. https://www.jianshu.com/p/2ed292ae2365
4. [GPG Hangs When Private Keys are Accessed](https://unix.stackexchange.com/questions/382279/gpg-hangs-when-private-keys-are-accessed)
5. [https://superuser.com/questions/1075404/how-can-i-restart-gpg-agent](https://superuser.com/questions/1075404/how-can-i-restart-gpg-agent)
