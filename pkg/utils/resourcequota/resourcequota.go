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
