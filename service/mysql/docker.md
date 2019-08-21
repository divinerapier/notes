# Run MySQL by Docker

``` sh
# run mysql:5.7
$ docker run --name some-mysql -e MYSQL_ALLOW_EMPTY_PASSWORD=true -p 33306:3306 -d mysql:5.7
You need to specify one of MYSQL_ROOT_PASSWORD, MYSQL_ALLOW_EMPTY_PASSWORD and MYSQL_RANDOM_ROOT_PASSWORD

# without password
$ docker run --name some-mysql -e MYSQL_ALLOW_EMPTY_PASSWORD=true -p 33306:3306 -d mysql:5.7

$ docker logs -f some-mysql
2019-08-08T03:13:45.886579Z 0 [ERROR] InnoDB: Write to file ./ibdata1failed at offset 0, 1048576 bytes should have been written, only 0 were written. Operating system error number 28. Check that your OS and file system support files of this size. Check also that the disk is not full or a disk quota exceeded.
2019-08-08T03:13:45.886596Z 0 [ERROR] InnoDB: Error number 28 means 'No space left on device'
2019-08-08T03:13:45.886603Z 0 [ERROR] InnoDB: Could not set the file size of './ibdata1'. Probably out of disk space
2019-08-08T03:13:45.886609Z 0 [ERROR] InnoDB: InnoDB Database creation was aborted with error Generic error. You may need to delete the ibdata1 file before trying to start up again.
2019-08-08T03:13:46.489727Z 0 [ERROR] Plugin 'InnoDB' init function returned error.
2019-08-08T03:13:46.489772Z 0 [ERROR] Plugin 'InnoDB' registration as a STORAGE ENGINE failed.
2019-08-08T03:13:46.489777Z 0 [ERROR] Failed to initialize builtin plugins.
2019-08-08T03:13:46.489780Z 0 [ERROR] Aborting

$ docker volume rm $(docker volume ls -qf dangling=true) 

# retry again
```


