# Create user and grant privileges

``` sql
> create user 'root'@'%' identified by '';
> grant all privileges on *.* to 'root'@'%' with grant option;
```
[also see](https://stackoverflow.com/questions/50177216/how-to-grant-all-privileges-to-root-user-in-mysql-8-0)
