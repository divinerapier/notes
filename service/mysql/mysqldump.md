# MySQLDump

## TL;DR

``` bash
$ tldr mysqldump

mysqldump

Backups MySQL databases.
More information: <https://dev.mysql.com/doc/refman/en/mysqldump.html>.

- Create a backup, user will be prompted for a password:
    mysqldump -u user --password database_name -r filename.sql

- Restore a backup, user will be prompted for a password:
    mysql -u user --password -e "source filename.sql" database_name

- Backup all databases redirecting the output to a file (user will be prompted for a password):
    mysqldump -u user -p --all-databases > filename.sql

- Restore all databases from a backup (user will be prompted for a password):
    mysql -u user -p < filename.sql
```

## Common Errors

### Unknown table 'COLUMN_STATISTICS' in information_schema (1109)

``` bash
$ mysqldump --column-statistics=0 -h 127.0.0.1 -P 3306 -u root -p database_name > database_name.sql
```

[For More](https://www.scommerce-mage.com/blog/mysqldump-throws-unknown-table-column_statistics-in-information_schema-1109.html)
