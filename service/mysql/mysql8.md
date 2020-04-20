# ERRORS

## MySQL8 Remote Access

``` bash
mysql> CREATE USER 'root'@'%' IDENTIFIED BY '123';
mysql> GRANT ALL PRIVILEGES ON *.* TO 'root'@'%';
```

## error 2059: Authentication plugin 'caching_sha2_password' cannot be loaded

``` bash
mysql> ALTER USER 'root'@'%' IDENTIFIED BY 'password' PASSWORD EXPIRE NEVER;
Query OK, 0 rows affected (0.41 sec)

mysql> ALTER USER 'root'@'%' IDENTIFIED WITH mysql_native_password BY 'password';
Query OK, 0 rows affected (0.41 sec)
```

## Reference

1. https://stackoverflow.com/questions/50570592/mysql-8-remote-access
1. https://blog.csdn.net/vkingnew/article/details/80105323
