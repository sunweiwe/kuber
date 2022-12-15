package handler

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/sunweiwe/kuber/pkg/api/kuber/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var _nodeHandler *NodeHandler

type NodeHandler struct {
	Client client.Client
	Log    logr.Logger
}

func (h *NodeHandler) Create(e event.CreateEvent, r workqueue.RateLimitingInterface) {
	dep, ok := e.Object.(*corev1.Node)
	if !ok {
		return
	}
	h.requeueTNetPol(dep.OwnerReferences, r)
}

func (h *NodeHandler) Update(e event.UpdateEvent, r workqueue.RateLimitingInterface) {
}

func (h *NodeHandler) Delete(e event.DeleteEvent, r workqueue.RateLimitingInterface) {
	dep, ok := e.Object.(*corev1.Node)
	if !ok {
		return
	}
	h.requeueTNetPol(dep.OwnerReferences, r)
}

func (h *NodeHandler) Generic(e event.GenericEvent, r workqueue.RateLimitingInterface) {
}

func newNodeHandler(c client.Client, log logr.Logger) *NodeHandler {
	if _nodeHandler != nil {
		return _nodeHandler
	}
	_nodeHandler = &NodeHandler{
		Client: c,
		Log:    log,
	}
	return _nodeHandler
}

func NewNodeHandler(c client.Client, log logr.Logger) *NodeHandler {
	return newNodeHandler(c, log)

}

func (h *NodeHandler) requeueTNetPol(owners []metav1.OwnerReference, r workqueue.RateLimitingInterface) {
	networks := v1beta1.TenantNetworkPolicyList{}
	if err := h.Client.List(context.Background(), &networks); err != nil {
		h.Log.Error(err, "failed to list tenant network policies")
		return
	}
	for _, tp := range networks.Items {
		r.Add(ctrl.Request{
			NamespacedName: client.ObjectKeyFromObject(&tp),
		})
	}
}
