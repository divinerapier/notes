# Docker

## Exec Scratch

``` bash
# Assumes scratch container is last launched one, else replace with container ID of
# scratch image, e.g. from `docker ps`, for example:
# scratch_container_id=401b31621b36
scratch_container_id=$(docker ps -ql)

docker run -d busybox:latest sleep 100
busybox_container_id=$(docker ps -ql)
docker cp "$busybox_container_id":/bin/busybox .

# The busybox binary will become whatever you name it (or the first arg you pass to it), for more info run:
# docker run busybox:latest /bin/busybox
# The `busybox --install` command copies the binary with different names into a directory.

docker cp ./busybox "$scratch_container_id":/busybox

docker exec -it "$scratch_container_id" /busybox sh -c '
export PATH="/busybin:$PATH"
/busybox mkdir /busybin
/busybox --install /busybin
sh'
```

https://stackoverflow.com/questions/54720824/how-to-docker-exec-a-container-built-from-scratch
