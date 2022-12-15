package handler

import (
	"github.com/go-logr/logr"
	"github.com/sunweiwe/kuber/pkg/api/kuber"
	"github.com/sunweiwe/kuber/pkg/api/kuber/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var _tenantResourceHandler *TenantResourceQuotaHandler

type TenantResourceQuotaHandler struct {
	Client client.Client
	Log    logr.Logger
}

func (h *TenantResourceQuotaHandler) Create(e event.CreateEvent, r workqueue.RateLimitingInterface) {
	rq, ok := e.Object.(*v1beta1.TenantResourceQuota)
	if !ok {
		return
	}
	h.requeueTenantResourceQuota(rq.Labels, r)
}

func (h *TenantResourceQuotaHandler) Update(e event.UpdateEvent, r workqueue.RateLimitingInterface) {
	current, currentOk := e.ObjectNew.(*v1beta1.TenantResourceQuota)
	before, beforeOk := e.ObjectOld.(*v1beta1.TenantResourceQuota)

	if !currentOk || !beforeOk {
		return
	}
	if !equality.Semantic.DeepEqual(current.Status, before.Status) {
		h.requeueTenantResourceQuota(current.Labels, r)
	}
}

func (h *TenantResourceQuotaHandler) Delete(e event.DeleteEvent, r workqueue.RateLimitingInterface) {
	rq, ok := e.Object.(*v1beta1.TenantResourceQuota)
	if !ok {
		return
	}
	h.requeueTenantResourceQuota(rq.Labels, r)
}

func (h *TenantResourceQuotaHandler) Generic(e event.GenericEvent, r workqueue.RateLimitingInterface) {

}

func newTenantResourceQuotaHandler(c *client.Client, log *logr.Logger) *TenantResourceQuotaHandler {
	if _tenantResourceHandler != nil {
		return _tenantResourceHandler
	}
	_tenantResourceHandler = &TenantResourceQuotaHandler{
		Client: *c,
		Log:    *log,
	}
	return _tenantResourceHandler
}

func NewTenantResourceQuotaHandler(c client.Client, log logr.Logger) *TenantResourceQuotaHandler {
	return newTenantResourceQuotaHandler(&c, &log)
}

func (h *TenantResourceQuotaHandler) requeueTenantResourceQuota(labels map[string]string, r workqueue.RateLimitingInterface) {
	tenantName, exist := labels[kuber.LabelTenant]
	if !exist {
		return
	}
	r.Add(ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name: tenantName,
		},
	})
}
