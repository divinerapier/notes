# Docker Compose

## cheatsheet
https://devhints.io/docker-compose

## Stop

``` sh
# stop all
$ docker-compose stop

# stop one or some
$ docker-compose stop service_name(not container's name) [...]
```

## Log Monitor

``` sh
$ while (true); do \sleep 1 && docker logs -f container_name; done;
```
