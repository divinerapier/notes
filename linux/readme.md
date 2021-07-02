1. 查看进程的线程数

``` bash
$ cat /proc/<pid>/status
```

2. 使用 `shell` 处理 `json` 数据并去重

``` bash
$ jq .vin audio.json | sed -e 's/^"//' -e 's/"$//' | sort | uniq
```
