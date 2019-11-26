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
总之，异步相较于多线程而言，有速度更快，占用资源更少的潜力。但是，操作系统本身就支持线程，使用线程开发也不需要特殊的编程模型。但是，异步编程需要编程语言提供库级别的支持。`Rust` 通过 `async fn` 创建一个返回 `Future` 的异步函数。
同时，传统的多线程模型就能提供比较好的效率，同时，`Rust` 占用内存小，行为可预测的特点使得即使不用 `async` 也能很好的工作。使用异步开发增加的额外复杂度并不总是值得的。

## Under the Hood: Excuting Futures and Tasks

本节介绍底层数据结构: `Future` 与异步任务是如何被调度的。

### The Future Trait

`Future` 是 `Rust` 异步编程的核心特性。一个 `Future` 是一个异步任务的计算单元，最终将产生一个值(可能为空`()`)。如下为一个简单的例子:

``` rust
trait SimpleFuture {
    type Output;
    fn poll(&mut self, wake: fn()) -> Poll<Self::Output>;
}

enum Poll<T> {
    Ready(T),
    Pending,
}
```

可以通过调用 `Poll` 来推动 `Future` 尽可能地接近完成。`Future` 完成，将返回 `Poll::Ready(Result)`。否则，将返回 `Poll::Pending`，并安排在 `Future` 准备好时调用 `wake()` 函数。 调用 `wake()` 时，执行器驱动 `Future` 再次调用 `poll`，推进 `Future` 完成。
如果离开 `wake()`，执行器将无法感知一个 `Future` 何时可以继续执行，将持续的轮询每一个 `Future`。正是通过 `wake()`，执行期可以精确的知道已经就绪可以调用 `poll` 的 `Futures` 。
例如，考虑以下情况：我们想从一个套接字中读取数据，该套接字可能已经或可能没有数据。 如果有数据，我们可以将其读入并返回 `Poll::Ready(data)`，但是如果没有数据可用，则我们的 `Future` 将受阻，无法再取得进展。 如果没有可用数据，则必须在套接字上的数据准备就绪时注册 `wake` 才能被调用，这将告诉执行者我们的 `Future` 已准备好继续执行。 一个简单的 `SocketRead` `Future` 可能看起来像这样：

``` Rust
pub struct SocketRead<'a> {
    socket: &'a Socket,
}

impl SimpleFuture for SocketRead<'_> {
    type Output = Vec<u8>;

    fn poll(&mut self, wake: fn()) -> Poll<Self::Output> {
        if self.socket.has_data_to_read() {
            // The socket has data-- read it into a buffer and return it.
            Poll::Ready(self.socket.read_buf())
        } else {
            // The socket does not yet have data.
            //
            // Arrange for `wake` to be called once data is available.
            // When data becomes available, `wake` will be called, and the
            // user of this `Future` will know to call `poll` again and
            // receive data.
            self.socket.set_readable_callback(wake);
            Poll::Pending
        }
    }
}
`