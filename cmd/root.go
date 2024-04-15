package cmd

import (
	"flag"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/superorbital/capargo/internal/controller"
	"github.com/superorbital/capargo/pkg/types"
	"istio.io/istio/pkg/cluster"
	"istio.io/istio/pkg/kube"
	istiolog "istio.io/istio/pkg/log"
	ctrl "sigs.k8s.io/controller-runtime"
	restconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

// Build information
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

// Flags
var loggingOptions = istiolog.DefaultOptions()
var (
	clusterID        string
	clusterNamespace string
	argoNamespace    string
	timeout          time.Duration
)

var rootCmd = &cobra.Command{
	Use:   "capargo",
	Short: "Runs the capargo controller",
	Run: func(cmd *cobra.Command, args []string) {
		// Get options for controller
		o := types.Options{
			ClusterID:        clusterID,
			ClusterNamespace: clusterNamespace,
			ArgoNamespace:    argoNamespace,
			Timeout:          timeout,
		}
		// Logger options
		istiolog.Configure(loggingOptions)
		logger := istiolog.RegisterScope("capargo-main", "")

		// Display build information
		b := BuildInfo{
			BuildTime: BuildTime,
			GitCommit: Revision,
			Version:   Version,
		}
		logger.Infof("Starting up capargo binary: version=%s, revision=%s, build time=%s",
			b.Version,
			b.GitCommit,
			b.BuildTime,
		)

		// Initialize controller
		config, err := restconfig.GetConfig()
		if err != nil {
			logger.Errorf("Failed to get restconfig: %v", err)
			os.Exit(1)
		}
		client, err := kube.NewClient(kube.NewClientConfigForRestConfig(config), cluster.ID(o.ClusterID))
		if err != nil {
			logger.Errorf("Unable to initialize Kubernetes client: %v", err)
			os.Exit(1)
		}
		ctx := ctrl.SetupSignalHandler()
		coll := controller.NewCollection(client)
		go coll.Synced().WaitUntilSynced(ctx.Done())
		if !client.RunAndWait(ctx.Done()) {
			logger.Error("Failed to start informers and sync client")
			client.Shutdown()
			os.Exit(1)
		}
		<-ctx.Done()
	},
}

func init() {
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	loggingOptions.AttachCobraFlags(rootCmd)
	rootCmd.Flags().StringVar(&clusterID, "id", "kind", "The name of the cluster where capargo is located.")
	rootCmd.Flags().StringVar(&clusterNamespace, "cluster-namespace", "", "The namespace to watch for clusters")
	rootCmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "The timeout period for any update action")
	rootCmd.Flags().StringVar(&argoNamespace, "argo-namespace", "", "The argo namespace in which to place the secrets")
	rootCmd.MarkFlagRequired("argo-namespace")
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
