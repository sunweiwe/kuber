//Package kube kube
package kube

import (
	"os"
	"path/filepath"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func AutoClientConfig() (*rest.Config, error) {

	if config, err := rest.InClusterConfig(); err != nil {
		home, _ := os.UserHomeDir()
		return clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube", "config"))
	} else {
		return config, nil
	}
}
