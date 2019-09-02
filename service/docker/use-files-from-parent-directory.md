# 使用父级目录文件

通常情况下，项目中的 `docker` 相关文件会在一个单独的目录，比如 `project/docker` 中，源码文件及其他会在 `project/src`, `project/config` 等中。所以，此时就希望在编译 `dockerfile` 时能够访问上级目录。

但是在 `docker` 中是不允许执行命令 `COPY ../ .` 的。此时，有两种方法解决问题。

## Docker 

执行命令 `docker build` 时，`dockerfile` 的工作目录即执行命令时的当前目录。所以，只需要在 `dockerfile` 期望的目录执行 `docker build -f path/to/your/dockerfile`。

## Docker-Compose

使用 `docker-compose` 时，无法使用上述方法解决。但是 `docker-compose` 中存在一个关键字 `context`，该字段的含义可以理解为工作路径，并且是相对于 `docker-compose.yml` 的相对路径(如果是相对路径的话)。

并且，无论在任何地方执行 `docker-compose -f path/to/docker-compose.yml` 都不会改变表现行为，都会使用 `context` 指定的路径作为工作路径。
