# Go Mod

## Disable checksum verify

使用本地仓库时，在 `sum.golang.org` 无法找到对应的 `checksum`, 可执行 
``` bash
$ export GOSUMDB=off
```
将检查关闭
