# Rust Async

## Getting Started

### Why Async

异步程序允许在相同 `CPU` 核心上，同时并发执行多个任务。在传统的多线程应用中，同时并发下载两个页面的代码应该是:

``` rust
fn get_two_sites() {
    // Spawn two threads to do work.
    let thread_one = thread::spawn(|| download("https://www.foo.com"));
    let thread_two = thread::spawn(|| download("https://www.bar.com"));

    // Wait for both threads to complete.
    thread_one.join().expect("thread one panicked");
    thread_two.join().expect("thread two panicked");
}
```
