package types

import "time"

type Options struct {
	ClusterID        string
	ClusterNamespace string
	ArgoNamespace    string
	Timeout          time.Duration
}
