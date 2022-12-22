package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TenantResourceQuotaSpec struct {
	// 租户在本集群的可以使用的总资源限制
	Hard corev1.ResourceList `json:"hard,omitempty"`
}

type TenantResourceQuotaStatus struct {
	// 租户在本集群的可以使用的总资源限制
	Hard corev1.ResourceList `json:"hard,omitempty"`
	// 已经申请了的资源
	Allocated corev1.ResourceList `json:"allocated,omitempty"`
	// 实际使用了的资源
	Used corev1.ResourceList `json:"used,omitempty"`

	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}

//+genclient
//+genclient:nonNamespaced
//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster,shortName=tquota,path=tenantresourcequotas
//+kubebuilder:subresource:status
//+kubebuilder:rbac:groups=gems,resources=TenantResourceQuota,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gems,resources=TenantResourceQuota/status,verbs=get;list;watch;create;update;patch;delete

type TenantResourceQuota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TenantResourceQuotaSpec   `json:"spec,omitempty"`
	Status TenantResourceQuotaStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TenantResourceQuotaList contains a list of TenantResourceQuota
type TenantResourceQuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TenantResourceQuota `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TenantResourceQuota{}, &TenantResourceQuotaList{})
}
