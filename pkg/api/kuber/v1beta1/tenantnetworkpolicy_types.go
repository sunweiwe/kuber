package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EnvironmentNetworkPolicy struct {
	Project string `json:"project,omitempty"`
	Name    string `json:"name,omitempty"`
}

type ProjectNetworkPolicy struct {
	Name string `json:"name,omitempty"`
}

type TenantNetworkPolicySpec struct {
	Tenant                     string                     `json:"tenant,omitempty"`
	TenantIsolated             bool                       `json:"tenantIsolated,omitempty"`
	ProjectNetworkPolicies     []ProjectNetworkPolicy     `json:"projectNetworkPolicies,omitempty"`
	EnvironmentNetworkPolicies []EnvironmentNetworkPolicy `json:"environmentNetworkPolicies,omitempty"`
}

type TenantNetworkPolicyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}

//+genclient
//+genclient:nonNamespaced
//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster,shortName=tnetpol
//+kubebuilder:subresource:status
//+kubebuilder:rbac:groups=gems,resources=TenantNetworkPolicy,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gems,resources=TenantNetworkPolicy/status,verbs=get;list;watch;create;update;patch;delete

type TenantNetworkPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TenantNetworkPolicySpec   `json:"spec,omitempty"`
	Status TenantNetworkPolicyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TenantNetworkPolicyList contains a list of TenantNetworkPolicy
type TenantNetworkPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TenantNetworkPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TenantNetworkPolicy{}, &TenantNetworkPolicyList{})
}
