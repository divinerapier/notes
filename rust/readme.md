# Rust Guide

## Installation

### Rustup

https://rustup.rs/

### Toolchains

``` bash
$ rustup default beta-2019-10-03
$ rustup component add cargo clippy llvm-tools-preview rust-analysis rust-docs rust-src rust-std rustc rustfmt
```

## IDE

### Vscode

#### rust-analyzer

[rust-analyzer](https://github.com/rust-analyzer/rust-analyzer) 目前比较好用的一个 `language server`，按照文档安装即可，如果需要在远端使用，在远端窗口手动安装插件即可。

[vscode-remote-extensionpack](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.vscode-remote-extensionpack)

优点: 类型推断更准确，能跳转更多的宏定义。   
缺点: 启动略慢，如果程序无法编译，可能会消耗大量内存来分析(最后结果就是 OOM 被干掉，很蠢)   

#### RLS

``` bash
$ rustup component add rls rust-analysis rust-src
```

[vscode-rls](https://marketplace.visualstudio.com/items?itemName=rust-lang.rust)
 
优点: 资源占用较小   
缺点: 能力比 `rust-analyzer` 弱   

### Clion

`Plugin` 搜索 `rust` 插件并安装。

优点: 跳转，代码提示都还不错。   
缺点: 用着不舒服。   

## Errors

### cargo watch not found

```
`cargo watch` failed with 127: /bin/sh: 1: cargo: not found
```

``` bash
$ cargo install cargo-watch
```

### linker `cc` not found

``` bash
$ cargo run
   Compiling hello-world v0.1.0 (/home/divinerapier/code/rust/github.com/divinerapier/example/hello-world)
error: linker `cc` not found
  |
  = note: No such file or directory (os error 2)

error: aborting due to previous error

error: could not compile `hello-world`.

To learn more, run the command again with --verbose.
```

please run:

``` bash
$ sudo apt install clang
```
