1. `API` 变动了，你对应的教程也跟着变啊，行不行。 
[示例代码](https://rust-lang.github.io/async-book/01_getting_started/04_async_await_primer.html)

``` rust
async fn async_main() {
    let f1 = learn_and_sing();
    let f2 = dance();

    // `join!` is like `.await` but can wait for multiple futures concurrently.
    // If we're temporarily blocked in the `learn_and_sing` future, the `dance`
    // future will take over the current thread. If `dance` becomes blocked,
    // `learn_and_sing` can take back over. If both futures are blocked, then
    // `async_main` is blocked and will yield to the executor.
    futures::join!(f1, f2);
}
```
一处 `API` 发生变动
```
futures::join!(f1, f2); -> futures::future::join(f1, f2).await; 
```
