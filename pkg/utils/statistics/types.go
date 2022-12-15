package statistics

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const ResourceLimitsPrefix = "limits."

func AddResourceList(total corev1.ResourceList, add corev1.ResourceList) {

}

type ResourceListCollectFunc func(corev1.ResourceName, *resource.Quantity, resource.Quantity)

func ResourceListCollect(into, values corev1.ResourceList, collect ResourceListCollectFunc) corev1.ResourceList {
	for resourceName, quantity := range values {
		lastQuantity := into[resourceName].DeepCopy()
		collect(resourceName, &lastQuantity, quantity)
		into[resourceName] = lastQuantity
	}

	return into
}
