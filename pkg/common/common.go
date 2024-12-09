package common

const (
	slug                       = "superorbital.io"
	ControllerName             = "capargo"
	ControllerNameLabel        = ControllerName + "." + slug + "/controller-name"
	ClusterNameAnnotation      = ControllerName + "." + slug + "/cluster-name"
	ClusterNamespaceAnnotation = ControllerName + "." + slug + "/cluster-namespace"
)
