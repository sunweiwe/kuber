package plugin

const (
	AnnotationIsPlugin = "plugins.kuber.io/is-plugin"

	// plugin category
	// example: kubernetes/security,core/network
	AnnotationCategory = "plugins.kuber.io/category"

	// description
	AnnotationPluginDescription = "plugins.kuber.io/description"
)

const (
	KuberNamespaceInstaller = "kuber-installer"
)
