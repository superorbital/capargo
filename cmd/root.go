package cmd

import (
	"flag"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/superorbital/capargo/internal/controller"
	"github.com/superorbital/capargo/pkg/types"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
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
var opts = zap.Options{}
var (
	clusterID        string
	clusterNamespace string
	argoNamespace    string
	timeout          time.Duration
)

// Scheme
var (
	scheme = runtime.NewScheme()
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
		logf.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
		logger := logf.Log.WithName("capargo-main")

		// Display build information
		b := BuildInfo{
			BuildTime: BuildTime,
			GitCommit: Revision,
			Version:   Version,
		}
		logger.Info("Starting up capargo binary",
			"version", b.Version,
			"revision", b.GitCommit,
			"build time", b.BuildTime,
		)

		// Initialize controller
		mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{
			Scheme: scheme,
		})
		if err != nil {
			logger.Error(err, "could not create manager")
			os.Exit(1)
		}

		err = builder.
			ControllerManagedBy(mgr).
			For(&clusterv1.Cluster{}).
			Owns(&corev1.Secret{}).
			Complete(&controller.ClusterKubeconfigReconciler{
				Client:  mgr.GetClient(),
				Options: o,
			})
		if err != nil {
			logger.Error(err, "could not create controller")
			os.Exit(1)
		}

		if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
			logger.Error(err, "could not start manager")
			os.Exit(1)
		}
	},
}

func init() {
	_ = corev1.AddToScheme(scheme)
	_ = clusterv1.AddToScheme(scheme)
	opts.BindFlags(flag.CommandLine)
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
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
