package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/superorbital/capargo/internal/controller"
	"istio.io/istio/pkg/cluster"
	"istio.io/istio/pkg/kube"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	restconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

var clientset *kubernetes.Clientset
var client kube.Client

func init() {
	var err error
	config := restconfig.GetConfigOrDie()
	clientcfg := kube.NewClientConfigForRestConfig(config)
	client, err = kube.NewClient(clientcfg, cluster.ID("kind-kind"))
	if err != nil {
		fmt.Printf("Unable to initialize Kubernetes client: %s\n", err)
		os.Exit(1)
	}
}

func main() {
	ctx := ctrl.SetupSignalHandler()
	coll := controller.NewCollection(client)
	go coll.Synced().WaitUntilSynced(ctx.Done())
	if !client.RunAndWait(ctx.Done()) {
		slog.Error("Failed to start informers and sync client")
		client.Shutdown()
		os.Exit(1)
	}
	<-ctx.Done()
}
