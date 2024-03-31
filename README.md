# capargo

Turn a Cluster API cluster into an ArgoCD cluster.

## Dev Dependencies

* [clusterctl](https://cluster-api.sigs.k8s.io/user/quick-start.html#install-clusterctl)
* [vcluster CLI](https://www.vcluster.com/docs/getting-started/setup)
* [argocd CLI](https://argo-cd.readthedocs.io/en/stable/cli_installation/)

## Demo 

Initialize a cluster once with `./hack/init-cluster.sh`.

Run the demo with `./hack/test-process.sh`

Clean up the demo with `./hack/cleanup-test.sh`