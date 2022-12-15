package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Service struct {
	ExtraLabels map[string]string `json:"extraLabels,omitempty"`
}

type Workload struct {
	// +kubebuilder:validation:Optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// +kubebuilder:validation:Optional
	ExtraLabels map[string]string `json:"extraLabels,omitempty"`
}

type TenantGatewayStatus struct {
	// ActAvailableReplicas nginx deployment 正常的pod数
	AvailableReplicas int32 `json:"availableReplicas"`
	// NodePort nginx service 占用的ports
	Ports []corev1.ServicePort `json:"ports"`
}

type Image struct {
	Repository string `json:"repository"`

	Tag string `json:"tag"`

	// +kubebuilder:validation:Enum=Never;Always;IfNotPresent
	PullPolicy string `json:"pullPolicy"`
}

type TenantGatewaySpec struct {
	// 负载均衡类型
	Type corev1.ServiceType `json:"type"` // NodePort or LoadBalancer
	// 负载均衡实例数
	Replicas *int32 `json:"replicas"`
	// 用以区分nginx作用域
	IngressClass string `json:"ingressClass"`
	// +kubebuilder:validation:Optional
	Image Image `json:"image"`
	// +kubebuilder:validation:Optional
	// +nullable
	Service *Service `json:"service"`
	// +kubebuilder:validation:Optional
	// +nullable
	Workload *Workload `json:"workload"`
}

//+genclient
//+genclient:nonNamespaced
//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster,shortName=tgw
//+kubebuilder:subresource:status
//+kubebuilder:rbac:groups=gems,resources=TenantGateway,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gems,resources=TenantGateway/status,verbs=get;list;watch;create;update;patch;delete

type TenantGateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TenantGatewaySpec   `json:"spec,omitempty"`
	Status TenantGatewayStatus `json:"status,omitempty"`
}
