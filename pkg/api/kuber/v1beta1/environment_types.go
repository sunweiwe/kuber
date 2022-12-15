package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DeletePolicy string

const (
	DeletePolicyNamespace = "deleteNamespace"

	DeletePolicyLabels = "deleteLabels"
)

// EnvironmentSpec defines the desired state of Environment
type EnvironmentSpec struct {
	// 租户
	Tenant string `json:"tenant"`
	// 项目
	Project string `json:"project"`
	// 关联的ns
	Namespace string `json:"namespace"`
	// 删除策略,选项为 delNamespace,delLabels
	DeletePolicy string `json:"deletePolicy"`
	// 资源限制
	ResourceQuota corev1.ResourceList `json:"resourceQuota,omitempty"`
	// 默认limitRange
	LimitRage []corev1.LimitRangeItem `json:"limitRange,omitempty"`
	// ResourceQuotaName
	ResourceQuotaName string `json:"resourceQuotaName,omitempty"`
	// LimitRageName
}

// EnvironmentStatus defines the observed state of Environment
type EnvironmentStatus struct {
	// 最后更新时间
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}

//+genclient
//+genclient:nonNamespaced
//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster,shortName=tenv,singular=environment
//+kubebuilder:subresource:status
//+kubebuilder:rbac:groups=gems,resources=Environment,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gems,resources=Environment/status,verbs=get;list;watch;create;update;patch;delete

type Environment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              EnvironmentSpec   `json:"spec,omitempty"`
	Status            EnvironmentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
type EnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Environment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Environment{}, &EnvironmentList{})
}
