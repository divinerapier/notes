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
这对于大多数程序来说，可以很好的工作。但也有一定的局限性，线程切换与线程之间共享数据会有很大的开销。即使，一个线程什么也不做，同样会消耗系统资源。消除这些开销，是设计异步程序模式的初衷。下面，使用 `async/await` 重写上面的代码。

``` rust
async fn get_two_sites_async() {
    // Create two different "futures" which, when run to completion,
    // will asynchronously download the webpages.
    let future_one = download_async("https://www.foo.com");
    let future_two = download_async("https://www.bar.com");

    // Run both futures to completion at the same time.
    join!(future_one, future_two);
}
```
