# capargo

>Automatically register any Cluster API cluster in your local ArgoCD instance.

The `capargo` controller (**C**luster **AP**I for **Argo**CD) performs the
tedious task of adding a new Cluster API cluster in the ArgoCD installation
that's running on the same cluster. This allows for a seamless experience
between the creation of a new cluster, and the cluster having all the necessary
workloads installed and maintained by ArgoCD

## Installation

TODO

## Development

### Pre-requisites

You will need the following tools installed in your computer for local development purposes:

 - [Golang](https://go.dev/doc/install)
 - [Taskfile](https://taskfile.dev/installation/)
 - [Docker](https://docs.docker.com/get-docker/)

 For testing your changes, you will additionally need:

 - [Kind](https://kind.sigs.k8s.io/)
 - [kubectl CLI](https://kubernetes.io/docs/tasks/tools/#kubectl)
 - [clusterctl CLI](https://cluster-api.sigs.k8s.io/user/quick-start.html#install-clusterctl)

### Building

To build the binary, simply run:

```sh
task build
```

To build the Docker image, run:

```sh
task build-image
```

### Testing

For testing changes, a Kind cluster with all the necessary components can be
bootstrapped by running the following task at the root of the repo:

```sh
task create-cluster
```

This will create the Kind cluster, a local container registry, and install
ArgoCD and Cluster API + Cluster API Provider vCluster (CAPV).

After this is done, you can retrieve the kubeconfig for the cluster using the command

```sh
task get-kubeconfig
```

which will create a `kind-cluster.kubeconfig` file that you can use to talk to
your cluster.

You will need to push your image of capargo to the local container registry:

```sh
docker push localhost:5001/superorbital/capargo:local
```

After which, you can run the demo script that will create a CAPV cluster for
your capargo Pod to interact with.

```sh
task install-demo
```

## Cleanup

To clean up the Kind cluster and the local registry:

```sh
task cleanup-cluster
```
