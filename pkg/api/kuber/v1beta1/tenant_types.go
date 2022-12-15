package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TenantSpec struct {
	// TenantName 租户名字
	TenantName string `json:"tenantName,omitempty"`
	// Admin 租户管理员列表
	Admin []string `json:"admin"`
	// Members 租户成员列表
	Members []string `json:"members,omitempty"`
}

type TenantStatus struct {
	// Environments 租户在本集群管控的环境
	Environments []string `json:"environments,omitempty"`
	// Namespaces 租户在本集群管控的namespace
	Namespaces []string `json:"namespaces,omitempty"`
	// LastUpdateTime 最后更新时间
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}

//+genclient
//+genclient:nonNamespaced
//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster,shortName=ten,singular=tenant
//+kubebuilder:subresource:status
//+kubebuilder:rbac:groups=gems,resources=Tenant,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gems,resources=Tenant/status,verbs=get;list;watch;create;update;patch;delete

type Tenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TenantSpec   `json:"spec,omitempty"`
	Status TenantStatus `json:"status,omitempty"`
}
