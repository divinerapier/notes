# Install tmux 2.9 on Centos

## install deps

### CentOS

``` bash
$ yum install gcc kernel-devel make ncurses-devel
```

### Ubuntu

``` bash
$ sudo apt install libncurses5-dev
```

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
./configure --prefix=/usr/local
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

## Local Build

``` bash
$ curl -LOk https://github.com/libevent/libevent/releases/download/release-2.1.8-stable/libevent-2.1.8-stable.tar.gz
$ curl -LOk https://github.com/tmux/tmux/releases/download/2.9/tmux-2.9.tar.gz
$ curl -LOk http://ftp.gnu.org/gnu/ncurses/ncurses-6.0.tar.gz

$ tar -xf tmux-2.9.tar.gz
$ tar -xf libevent-2.1.8-stable.tar.gz
$ tar -xf ncurses-6.0.tar.gz

$ cd libevent-2.1.8-stable; ./configure --prefix=$HOME/.local/; make -j32; make install; cd -
$ cd ncurses-6.0; ./configure --prefix=$HOME/.local/; make -j32; make install; cd -
$ cd tmux-2.9; CFLAGS="-I$HOME/.local/include -I$HOME/.local/include/ncurses " LDFLAGS="-L$HOME/.local/lib" ./configure --prefix=$HOME/.local
$ make -j32; make install
```

### Tips

编译 `ncurses` 可能会遇到如下错误:

``` 
from ../ncurses/lib_gen.c:19:
_26956.c:843:15: error: expected ‘)’ before ‘int’
../include/curses.h:1631:56: note: in definition of macro ‘mouse_trafo’
 #define mouse_trafo(y,x,to_screen) wmouse_trafo(stdscr,y,x,to_screen)
                                                        ^
```

打开文件 `include/curses.h`, 找到

``` c
extern NCURSES_EXPORT(bool)    mouse_trafo (int*, int*, bool);              /* generated */
``` 
删除 `/* generated */`
