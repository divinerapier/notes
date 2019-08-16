# Google File System

## 架构 

包含一个 `master server`，若干 `chunk server` 与 `client`。

### Master Server

保存文件 `metadata`。`client` 不会从/向 `master server` 读/写数据。

以读取文件为例:  
1. `client` 根据 `offset` 计算目标 `chunk` 在文件所有 `chunks` 中的 `index`
2. `client` 请求 `master` rpc   
  > request: filename, chunk index   
  > response: chunk handle, location   
3. `client` 缓存信息:   
  (filename, chunk index) -> (chunk handle, location)   
4. `client` 从所有 `replicas` 选择一个读取文件

### Chunk Size

`GFS` 设计 `chunk size` 为 `64 M`。

#### Advantage


#### Disadvantage
