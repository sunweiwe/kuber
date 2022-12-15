package handler

import (
	"github.com/go-logr/logr"
	"github.com/sunweiwe/kuber/pkg/api/kuber/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

/*
	监听所有的环境事件，当环境租户和namespace变更的时候,让对应的租户重新计算状态
*/
var _environmentHandler *EnvironmentHandler

type EnvironmentHandler struct {
	Client client.Client
	Log    logr.Logger
}

func (h *EnvironmentHandler) Create(e event.CreateEvent, r workqueue.RateLimitingInterface) {
	env, ok := e.Object.(*v1beta1.Environment)
	if !ok {
		return
	}

	r.Add(ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name: env.Spec.Tenant,
		},
	})
}

func (h *EnvironmentHandler) Update(e event.UpdateEvent, r workqueue.RateLimitingInterface) {
	current, currentOk := e.ObjectNew.(*v1beta1.Environment)
	before, beforeOk := e.ObjectOld.(*v1beta1.Environment)

	if !currentOk || !beforeOk {
		return
	}
	if current.Spec.Tenant == before.Spec.Tenant && current.Spec.Namespace == before.Spec.Namespace {
		return
	}
	r.Add(ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name: current.Spec.Tenant,
		},
	})
	r.Add(ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name: before.Spec.Tenant,
		},
	})
}

func (h *EnvironmentHandler) Delete(e event.DeleteEvent, r workqueue.RateLimitingInterface) {
	env, ok := e.Object.(*v1beta1.Environment)
	if !ok {
		return
	}
	r.Add(ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name: env.Spec.Tenant,
		},
	})
}

func (h *EnvironmentHandler) Generic(e event.GenericEvent, r workqueue.RateLimitingInterface) {}

func newEnvironmentHandler(c *client.Client, log *logr.Logger) *EnvironmentHandler {
	if _environmentHandler != nil {
		return _environmentHandler
	}
	_environmentHandler = &EnvironmentHandler{
		Client: *c,
		Log:    *log,
	}

	return _environmentHandler
}

func NewEnvironmentHandler(c client.Client, log logr.Logger) *EnvironmentHandler {
	return newEnvironmentHandler(&c, &log)
}
