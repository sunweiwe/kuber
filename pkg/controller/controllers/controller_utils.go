package controllers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// event reason
const (
	ReasonFailedCreateSubResource = "FailedCreateSubResource"
	ReasonFailedCreate            = "FailedCreate"
	ReasonFailedDelete            = "FailedDelete"
	ReasonFailedUpdate            = "FailedUpdate"
	ReasonCreated                 = "Created"
	ReasonDeleted                 = "Deleted"
	ReasonUpdated                 = "Updated"
	ReasonUnknownError            = "UnknownError"
)

func ExistOwnerRef(meta metav1.ObjectMeta, owner metav1.OwnerReference) bool {
	var exist bool
	for _, ref := range meta.OwnerReferences {
		if ref.APIVersion == owner.APIVersion && ref.Kind == owner.Kind && ref.Name == owner.Name {
			exist = true
			break
		}
	}
	return exist
}
