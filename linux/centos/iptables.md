# IPTABLES
## 开放指定端口
``` zsh
$ sudo iptables -I INPUT -i eth0 -p tcp -dport 22 -j ACCEPT
```