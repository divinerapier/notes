# 系统配置
## CentOS 7 首次登陆
```
Initial setup of CentOS Linux 7 (Core)
1) [!] License information
	(License not accepted)
Please make your choice from [ '1' to enter the License information spoke | 'q' to quit | 'c' to continue | 'r' to refresh]:
依次输入: 1, 2, q, yes
```

## 默认登陆界面
### 删除原有登陆配置
``` zsh
$ rm -f /etc/systemd/system/default.target
```
### 设置默认使用 multi-user
``` zsh
$ sudo ln -sf /lib/systemd/system/runlevel3.target /etc/systemd/system/default.target
```
<i>或者</i>
``` zsh
$ sudo ln -sf /lib/systemd/system/multi-user.target /etc/systemd/system/default.target
```
<i>或者</i>
``` zsh
$ sudo systemctl set-default multi-user.target
```
### 设置默认使用 graphical
``` zsh
$ sudo ln -sf /lib/systemd/system/runlevel5.target /etc/systemd/system/default.target
```
<i>或者</i>
``` zsh
$ sudo ln -sf /lib/systemd/system/graphical.target /etc/systemd/system/default.target
```
<i>或者</i>
``` zsh
$ sudo systemctl set-default graphical.target
```

## 普通用户无法使用sudo
```zsh
su -
chmod 777 /etc/sudoers
echo "Username  ALL=(ALL)   ALL" >> /etc/sudoers
chmod 440 /etc/sudoers
exit
```

# 常用软件配置

## Zsh
### 下载Zsh
``` zsh
$ sudo yum install -y zsh
```
### 下载 Oh-my-zsh
``` zsh
$ sh -c "$(curl -fsSL https://raw.githubusercontent.com/robbyrussell/oh-my-zsh/master/tools/install.sh)"
```
<i>或者</i>
``` zsh
$ sh -c "$(wget https://raw.githubusercontent.com/robbyrussell/oh-my-zsh/master/tools/install.sh -O -)"
```
### 设置Zsh为默认Shell
``` zsh
$ chsh -s /bin/zsh
```
然后,重启计算机。

## Git
### 安装
``` zsh
$ sudo yum install -y git
```
### 配置Git
``` zsh
$ git config --global user.name "DivineRapier"
$ git config --global user.email DivineRapier@example.com
$ git config --global http.sshVerify false
$ git config --global core.editor vim
$ git config --global merge.tool vimdiff
```
### 查看配置
查看已有的全部配置
``` zsh
$ git config --list
user.name "DivineRapier"
user.email DivineRapier@example.com
http.sshVerify false
core.editor vim
merge.tool vimdiff
```
查看指定变量配置
``` zsh
$ git config user.name
DivineRapier
```

## openssh
### 安装
``` zsh
$ sudo yum install -y openssh-clients openssh-server
```