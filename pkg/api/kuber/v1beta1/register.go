package v1beta1

import (
	"github.com/sunweiwe/kuber/pkg/api/kuber"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var SchemeGroupVersion = schema.GroupVersion{Group: kuber.GroupName, Version: "v1beta1"}

var (
	GroupVersion = SchemeGroupVersion

	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)

var (
	SchemeTenant      = GroupVersion.WithKind("Tenant")
	SchemeEnvironment = GroupVersion.WithKind("Environment")
)
