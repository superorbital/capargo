package cmd

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/superorbital/capargo/internal/controller"
	"github.com/superorbital/capargo/pkg/common"
	"github.com/superorbital/capargo/pkg/types"

	corev1 "k8s.io/api/core/v1"

	apimachinerytypes "k8s.io/apimachinery/pkg/types"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	clientconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
	workers          int
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
		mgr, err := manager.New(clientconfig.GetConfigOrDie(), manager.Options{
			Scheme: scheme,
			Controller: config.Controller{
				MaxConcurrentReconciles: workers,
			},
		})
		if err != nil {
			logger.Error(err, "could not create manager")
			os.Exit(1)
		}

		err = builder.
			ControllerManagedBy(mgr).
			For(&clusterv1.Cluster{}).
			Watches(&corev1.Secret{},
				handler.EnqueueRequestsFromMapFunc(
					func(ctx context.Context, obj client.Object) []reconcile.Request {
						s := obj.(*corev1.Secret)
						if _, ok := s.Labels[common.ControllerNameLabel]; ok {
							var name string
							var namespace string
							if name, ok = s.Annotations[common.ClusterNameAnnotation]; !ok {
								return nil
							}
							if namespace, ok = s.Annotations[common.ClusterNamespaceAnnotation]; !ok {
								return nil
							}
							return []reconcile.Request{
								{
									NamespacedName: apimachinerytypes.NamespacedName{
										Name:      name,
										Namespace: namespace,
									},
								},
							}
						}
						return nil
					})).
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
	rootCmd.Flags().IntVar(&workers, "workers", 3, "The number of concurrent workers available to reconcile the state.")
	rootCmd.Flags().StringVar(&clusterNamespace, "cluster-namespace", "", "The namespace to watch for clusters.")
	rootCmd.Flags().DurationVar(&timeout, "timeout", 5*time.Minute, "The timeout period for any update action.")
	rootCmd.Flags().StringVar(&argoNamespace, "argo-namespace", "", "The argo namespace in which to place the secrets.")
	rootCmd.MarkFlagRequired("argo-namespace")
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
