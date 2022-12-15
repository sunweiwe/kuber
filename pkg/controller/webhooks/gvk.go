package webhooks

import (
	"github.com/sunweiwe/kuber/pkg/api/kuber/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	gkvTenant = metav1.GroupVersionKind{
		Group:   v1beta1.GroupVersion.Group,
		Version: v1beta1.GroupVersion.Version,
		Kind:    "tenant",
	}

	gkvTenantResourceQuota = metav1.GroupVersionKind{
		Group:   v1beta1.GroupVersion.Group,
		Version: v1beta1.GroupVersion.Version,
		Kind:    "TenantResourceQuota",
	}

	gkvTenantNetworkPolicy = metav1.GroupVersionKind{
		Group:   v1beta1.GroupVersion.Group,
		Version: v1beta1.GroupVersion.Version,
		Kind:    "TenantNetworkPolicy",
	}
)
