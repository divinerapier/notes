# apt-get 错误
``` zsh
Fetched 94.5 kB in 1s (48.6 kB/s)
E: Could not get lock /var/lib/dpkg/lock - open (11: Resource temporarily unavailable)
E: Unable to lock the administration directory (/var/lib/dpkg/), is another process using it?
```
解决办法
``` zsh
sudo rm /var/cache/apt/archives/lock
sudo rm /var/lib/dpkg/lock
```