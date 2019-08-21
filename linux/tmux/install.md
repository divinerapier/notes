# Install tmux 2.9 on Centos

## install deps
yum install gcc kernel-devel make ncurses-devel

## libevent

``` sh
curl -LOk https://github.com/libevent/libevent/releases/download/release-2.1.8-stable/libevent-2.1.8-stable.tar.gz
tar -xf libevent-2.1.8-stable.tar.gz
cd libevent-2.1.8-stable
./configure --prefix=/usr/local
make -j4
make install
```

## tmux

``` sh
curl -LOk https://github.com/tmux/tmux/releases/download/2.9/tmux-2.9.tar.gz
tar -xf tmux-2.9.tar.gz
cd tmux-2.9
LDFLAGS="-L/usr/local/lib -Wl,-rpath=/usr/local/lib" ./configure --prefix=/usr/local
make -j4
make install
```

## add to path

``` sh
echo "export PATH=/usr/local/bin:$PATH" >> ~/.bashrc
```

## check version

``` sh
tmux -V
```
