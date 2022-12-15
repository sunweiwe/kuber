//Package kuber for common constant
package kuber

const (
	LabelTenant      = GroupName + "/tenant"
	LabelProject     = GroupName + "/project"
	LabelEnvironment = GroupName + "/environment"
	LabelApplication = GroupName + "/application"
	LabelZone        = GroupName + "/zone"
	LabelPlugins     = GroupName + "/plugins"

	NamespaceSystem    = "kuber"
	NamespaceLocal     = "kuber-local"
	NamespaceInstaller = "kuber-installer"
	NamespaceMonitor   = "kuber-monitoring"
	NamespaceLogging   = "kuber-logging"
	NamespaceGateway   = "kuber-gateway"
	NamespaceEvent     = "kuber-event"
	NamespaceWorkflow  = "kuber-workflow-system"
)

const (
	FinalizerNamespace     = "finalizer." + GroupName + "/namespace"
	FinalizerResourceQuota = "finalizer." + GroupName + "/resourcequota"
	FinalizerGateway       = "finalizer." + GroupName + "/gateway"
	FinalizerNetworkPolicy = "finalizer." + GroupName + "/netWorkPolicy"
	FinalizerLimitRange    = "finalizer." + GroupName + "/limitRange"
	FinalizerEnvironment   = "finalizer." + GroupName + "/environment"
)
