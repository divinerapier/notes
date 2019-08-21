# MariaDB

## 下载

### CentOS

``` zsh
# 安装 mariadb
$ sudo yum install -y mariadb mariadb-server
```

### MacOS

``` zsh
# 安装 mysql
$ brew install mysql
```

> We've installed your MySQL database without a root password. To secure it run:
> `mysql_secure_installation`
> MySQL is configured to only allow connections from localhost by default
> To connect run:
> `mysql -uroot`
> To have launchd start mysql now and restart at login:
> `brew services start mysql`
> Or, if you don't want/need a background service you can just run:
> `mysql.server start`

## 启动

### CentOS

``` zsh
# 启动MariaDB
$ sudo systemctl start mariadb

# 停止MariaDB
$ sudo systemctl stop mariadb

# 重启MariaDB
$ sudo systemctl restart mariadb

# 设置开机启动
$ sudo systemctl enable mariadb
```

### MacOS

``` zsh
# 启动MySQL(非后台)
$ mysql.server start

# 开机启动
$ brew services start mysql
```

## 首次启动设置密码

在启动 MariaDB(MySQL)之后

``` zsh
# 初始化
$ mysql_secure_installation
```

``` zsh
 Enter current password for root (enter for none):  输入当前的root密码(默认空)，直接回车

 Set root password? [Y/n] 设置新密码

 Remove anonymous users? [Y/n] 移除匿名用户

 Disallow root login remotely? [Y/n] 禁止root用户远程登录

 Remove test database and access to it? [Y/n] 移除测试数据库

 Reload privilege tables now? [Y/n]
```

## 登陆授权

在设置 root 用户密码之后，登陆以授权

``` zsh
# 使用 root 登陆
$ mysql -uroot -p
```

``` sql
-- 使用 root 身份创建新的用户并授权
MariaDB [(none)]> grant all privileges on *.* to username@'%'identified by 'password';
```