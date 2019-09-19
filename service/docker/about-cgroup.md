# 关于 cgroup

本次技术研究源于 `container oom` 错误。

首先，查阅了一些关于 `cgroup`，`docker oom` 相关的文档，看一些别人遇到过，解决过哪些问题及原因。然后，分别用 `c`, `go` 写了一个 `poc` 来看一下 `oom` 时的表现。

## Code 

``` c
#include <stdlib.h>
#include <stdio.h>
#include <unistd.h>

int main() {
    sleep(1);
    int count = 0;
    while (1) {
        char * array = (char*)malloc(sizeof(char) * 1<<20);
        if (array == NULL) {
            printf("loop times: %d. failed to allocates\n", count);
            return 1;
        }
        count+=1;
        if (count % 100 == 0) {
            printf("loop times: %d is ok\n", count);
        }
    }
    return 0;
}
```

``` go
package main

import (
	"fmt"
	"time"
)

func main() {
    time.Sleep(time.Second)
    array := make([][]byte, 0)
    count := 0
    for {
        current := make([]byte, 2<<20)
        if current == nil {
            fmt.Printf("failed to allocate memory.")
            return
        }
        count += 1
        array = append(array, current)
        if count%1000 == 0 {
            fmt.Printf("times: %d is ok\n", count)
        }
    }
}
```
## Dockerfile

``` dockerfile
FROM gcc:4.9

COPY . /usr/src/myapp
WORKDIR /usr/src/myapp
RUN ls /usr/src/
RUN ls /usr/src/myapp
RUN gcc -o myapp main.c
```

``` dockerfile
FROM golang:1.13

COPY . /usr/local/go/src/myapp
WORKDIR /usr/local/go/src/myapp
RUN ls /usr/local/go/src/myapp
RUN go build -o myapp main.go
```

## Run

``` bash
# --oom-kill-disable: 为可选参数，两个镜像，是否有 '--oom-kill-disable' 共四个测试用例

# 启动容器，设置内存限制
$ docker container run -d --memory=5m --memory-swap=10m [--oom-kill-disable] image-cgroup-c sleep 1000000000000
# 进入容器执行
$ ./myapp

# 同上
$ docker container run -d --memory=5m --memory-swap=10m [--oom-kill-disable] image-cgroup-go sleep 1000000000000
```

## Phenomenon

1. c 
  执行一段时间，会直接退出容器。
2. c --oom-kill-disable
  执行一段时间，程序停住。如果同时执行两个进程，在都停住后，杀死一个，另一个会继续执行。
3. go
  执行一段时间，会结束进程(未退出容器)。出现过两种错误: `Killed`, `Segmentation fault`。没有出现过 `panic`。
4. go --oom-kill-disable
  同 `c --oom-kill-disable`
 

)
