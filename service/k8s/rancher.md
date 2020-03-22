# Kubernetes On Rancher

## Install Rancher On A Single Node

### Install Rancher By Docker

``` bash
export LOCAL_HTTP_PORT=80
export LOCAL_HTTPS_PORT=443

$ docker run -d --restart=unless-stopped \
    -p ${LOCAL_HTTP_PORT}:80 -p ${LOCAL_HTTPS_PORT}:443 \
    rancher/rancher:latest
```

## Install Kubenetes On Rancher

### Configure Rancher

export HOST=192.168.1.100

1. 打开网址 `https://${HOST}:${LOCAL_HTTPS_PORT}`
2. 设置 `admin` 账户的密码. `(password)`
3. 创建 `Clustr`
4. `cni` 请选择 `calico`

### Install Kubernetes

``` bash
$ sudo docker run -d --privileged --restart=unless-stopped --net=host -v /etc/kubernetes:/etc/kubernetes -v /var/run:/var/run rancher/rancher-agent:v2.3.5 --server https://${HOST}:${LOCAL_HTTPS_PORT} --token ${TOKEN} --ca-checksum ${CHECK_SUM} --etcd --controlplane --worker
```

### Copy Kubeconfig

1. 选择 `Cluster`
2. 点击 `Kubeconfig File`
3. 复制到本地文件 `~/.kube/config`

### Install Kubelet

``` bash
$ curl -LO https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/amd64/kubectl
$ chmod +x ./kubelet
$ sudo mv ./kubelet /usr/local/bin/kubelet
```

### Verify Kubelet

``` bash
$ kubectl cluster-info
```

## References

1. https://rancher.com/docs/rancher/v2.x/en/installation/other-installation-methods/single-node-docker/
1. https://kubernetes.io/docs/tasks/tools/install-kubectl/#install-kubectl-on-linux
1. https://kubernetes.io/docs/tasks/tools/install-kubectl/#verifying-kubectl-configuration
1. https://kubernetes.io/docs/tasks/tools/install-kubectl/#optional-kubectl-configurations
