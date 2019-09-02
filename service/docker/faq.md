### standard_init_linux.go:195: exec user process caused "no such file or directory"

使用 `alpine:3.8` 作为底包，修改成 `centos:7` 即可修复。推测是 `alpine` 缺少必要的动态链接库。
