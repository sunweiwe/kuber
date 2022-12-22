package controllers

import (
	"context"

	"github.com/sunweiwe/kuber/pkg/api/kuber"
	"github.com/sunweiwe/kuber/pkg/api/kuber/v1beta1"
	"github.com/sunweiwe/kuber/pkg/utils/statistics"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type TenantResourceQuotaReconciler struct {
	client.Client
}

//+kubebuilder:rbac:groups=go.kuber.io,resources=tenantresourcequotas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=go.kuber.io,resources=tenantresourcequotas/status,verbs=get;update;patch

func (r *TenantResourceQuotaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		调度逻辑:
		计算总资源，筛选所有关联的ResourceQuota，将资源加起来就是使用的和申请的
	*/
	log := ctrl.LoggerFrom(ctx)
	log.Info("reconciling...")
	defer log.Info("reconcile done")

	var rq v1beta1.TenantResourceQuota
	if err := r.Get(ctx, req.NamespacedName, &rq); err != nil {
		log.Error(err, "get resource quota")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var resourceQuotaList corev1.ResourceQuotaList
	if err := r.List(ctx, &resourceQuotaList,
		client.MatchingLabels{kuber.LabelTenant: rq.Name},
		client.InNamespace(metav1.NamespaceAll),
	); err != nil {
		log.Error(err, "list resource quota")
		return ctrl.Result{}, nil
	}

	emptyResources := corev1.ResourceList{}
	for name := range rq.Spec.Hard {
		emptyResources[name] = resource.MustParse("0")
	}

	used, hard := emptyResources.DeepCopy(), emptyResources.DeepCopy()
	for _, item := range resourceQuotaList.Items {
		statistics.ResourceList(used, item.Status.Used)
		statistics.ResourceList(hard, item.Status.Hard)
	}

	hard, used = fixInvalidResourceName(hard), fixInvalidResourceName(used)

	if !equality.Semantic.DeepEqual(rq.Status.Used, used) || !equality.Semantic.DeepEqual(rq.Status.Allocated, hard) {
		log.Info("updating status")
		rq.Status.LastUpdateTime = metav1.Now()
		rq.Status.Used = used
		rq.Status.Allocated = hard
		rq.Status.Hard = hard
		if err := r.Status().Update(ctx, &rq); err != nil {
			log.Error(err, "update resource quota status")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func fixInvalidResourceName(list corev1.ResourceList) corev1.ResourceList {
	if v, ok := list["request.storage"]; ok {
		list["limits.storage"] = v.DeepCopy()
	}

	return list
}

func (r *TenantResourceQuotaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.TenantResourceQuota{}).
		Watches(&source.Kind{Type: &corev1.ResourceQuota{}}, NewResourceQuotaHandler()).
		Complete(r)
}

/*
	监听所有的ResourceQuota事件，当ResourceQuota变更的时候,让对应的TenantResourceQuota重新计算
*/

func NewResourceQuotaHandler() handler.Funcs {

	return handler.Funcs{
		CreateFunc: func(e event.CreateEvent, r workqueue.RateLimitingInterface) {
			requeueTenantResourceQuota(e.Object, r)
		},
		UpdateFunc: func(e event.UpdateEvent, r workqueue.RateLimitingInterface) {
			currentRq, currentOk := e.ObjectNew.(*corev1.ResourceQuota)
			beforeRq, beforeOk := e.ObjectOld.(*corev1.ResourceQuota)
			if !currentOk || !beforeOk {
				return
			}

			if !equality.Semantic.DeepEqual(currentRq.Status, beforeRq.Status) {
				requeueTenantResourceQuota(e.ObjectNew, r)
			}
		},
		DeleteFunc: func(de event.DeleteEvent, rli workqueue.RateLimitingInterface) {
			requeueTenantResourceQuota(de.Object, rli)
		},
	}
}

func requeueTenantResourceQuota(obj client.Object, r workqueue.RateLimitingInterface) {
	Labels := obj.GetLabels()
	if Labels == nil {
		return
	}
	if tenantName := Labels[kuber.LabelTenant]; tenantName != "" {
		r.Add(ctrl.Request{NamespacedName: types.NamespacedName{Name: tenantName}})
	}

}
