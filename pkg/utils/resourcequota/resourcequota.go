//Package resourcequota utils
package resourcequota

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

/*
租户级别限制:
cpu memory storage

环境级别限制
cpu memory storage
pods deployments statefulsets services configmaps secrets jobs cronjobs persistentvolumeclaims
*/

var (
	ResourceDeployments            corev1.ResourceName = "count/deployments.apps"
	ResourceStatefulSets           corev1.ResourceName = "count/statefulsets.apps"
	ResourceJobs                   corev1.ResourceName = "count/jobs.batch"
	ResourceCronJobs               corev1.ResourceName = "count/cronjobs.batch"
	ResourceSecrets                corev1.ResourceName = "count/secrets"
	ResourceConfigMaps             corev1.ResourceName = "count/configmaps"
	ResourceServices               corev1.ResourceName = "count/services"
	ResourcePersistentVolumeClaims corev1.ResourceName = "count/persistentvolumeclaims"
	ResourceDaemonsets             corev1.ResourceName = "count/daemonsets.apps"
	ResourceIngresses              corev1.ResourceName = "count/ingresses.extensions"
)

const (
	DefaultResourceQuotaName = "default"
	DefaultLimitRangeName    = "default"
)

func GetDefaultTenantResourceQuota() corev1.ResourceList {
	return corev1.ResourceList{
		corev1.ResourceLimitsCPU:       resource.MustParse("0"),
		corev1.ResourceLimitsMemory:    resource.MustParse("0Gi"),
		corev1.ResourceRequestsStorage: resource.MustParse("0Gi"),
	}
}
