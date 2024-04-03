package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/superorbital/capargo/internal/controller"
	"istio.io/istio/pkg/cluster"
	"istio.io/istio/pkg/kube"
	ctrl "sigs.k8s.io/controller-runtime"
	restconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	Version   = devVersion
	BuildTime = unknown
	Revision  = unknown
)

const unknown = "unknown"
const devVersion = "dev"

type BuildInfo struct {
	Version   string
	GitCommit string
	BuildTime string
}

func main() {
	b := BuildInfo{
		BuildTime: BuildTime,
		GitCommit: Revision,
		Version:   Version,
	}
	fmt.Printf("Starting up capargo binary\n%+v\n", b)
	config := restconfig.GetConfigOrDie()
	clientcfg := kube.NewClientConfigForRestConfig(config)
	client, err := kube.NewClient(clientcfg, cluster.ID("kind-kind"))
	if err != nil {
		fmt.Printf("Unable to initialize Kubernetes client: %s\n", err)
		os.Exit(1)
	}
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
