# Minikube

## Install

``` sh
$ brew install minikube
```

## Start

### On MaxOS

``` sh
$ DOCKER_HOST=/var/run/docker.sock minikube --vm-driver=none start \
  --extra-config=kubelet.resolv-conf=/run/systemd/resolve/resolv.conf
```

## Stop

### Stop

``` sh
$ minikube stop
```

### Delete

``` sh
$ minikube delete
```

## Logs

``` sh
$ minikube logs
```
