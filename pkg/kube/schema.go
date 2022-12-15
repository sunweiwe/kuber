package kube

import "k8s.io/apimachinery/pkg/runtime"

func ConfigureSchema(schema *runtime.Scheme) {

}

func Scheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	ConfigureSchema(scheme)
	return scheme
}
