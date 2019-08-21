# Official Document About Cluster
# 官方文档
[点击](https://coreos.com/etcd/docs/latest/op-guide/clustering.html) 查看集群相关官方文档   
``` zsh
usage: etcd [flags]
       start an etcd server

       etcd --version
       show the version of etcd

       etcd -h | --help
       show the help information about etcd

       etcd --config-file
       path to the server configuration file


clustering flags:

        --initial-advertise-peer-urls 'http://localhost:2380'
                list of this member's peer URLs to advertise to the rest of the cluster.
        --initial-cluster 'default=http://localhost:2380'
                initial cluster configuration for bootstrapping.
        --initial-cluster-state 'new'
                initial cluster state ('new' or 'existing').
        --initial-cluster-token 'etcd-cluster'
                initial cluster token for the etcd cluster during bootstrap.
                Specifying this can protect you from unintended cross-cluster interaction when running multiple clusters.
        --advertise-client-urls 'http://localhost:2379'
                list of this member's client URLs to advertise to the public.
                The client URLs advertised should be accessible to machines that talk to etcd cluster. etcd client libraries parse these URLs to connect to the cluster.
        --discovery ''
                discovery URL used to bootstrap the cluster.
        --discovery-fallback 'proxy'
                expected behavior ('exit' or 'proxy') when discovery services fails.
                "proxy" supports v2 API only.
        --discovery-proxy ''
                HTTP proxy to use for traffic to discovery service.
        --discovery-srv ''
                dns srv domain used to bootstrap the cluster.
        --strict-reconfig-check
                reject reconfiguration requests that would cause quorum loss.
        --auto-compaction-retention '0'
                auto compaction retention in hour. 0 means disable auto compaction.

```