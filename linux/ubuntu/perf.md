# Install Perf

## On Ubuntu 16.04

### Installation

``` bash
$ uname -a
Linux mtt 4.15.0-60-generic #67~16.04.1-Ubuntu SMP Mon Aug 26 08:57:33 UTC 2019 x86_64 x86_64 x86_64 GNU/Linux

$ sudo apt install linux-tools-common
$ sudo apt install linux-tools-generic linux-tools-4.15.0-60-generic
```

### Enabling perf for use by unprivileged users 

``` bash
$ echo -1 | sudo tee /proc/sys/kernel/perf_event_paranoid
```

## Rust

``` bash
$ cargo install flamegraph
```

## Reference

+ [Install perf on linux](https://stackoverflow.com/questions/39456308/i-cant-run-the-perf-command-perf-is-a-linux-profiler-for-stack-traces)
+ [Rust flamegraph](https://github.com/ferrous-systems/flamegraph)
