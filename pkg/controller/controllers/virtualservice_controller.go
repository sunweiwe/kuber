package controllers

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// ServiceEntryReconciler 用于为对开启 虚拟域名的 namespace 中service创建与虚拟域名相同的 serviceentries
// 功能：
// 1. 观察 namespace 是否具有虚拟空间标志 annotation "kuber.io/virtualdomain={virtualdomain name}"
// 2. 若有，则为该namespace下的service创建一个virtual service，并设置其hosts 为 {serviceName}.{virtualServiceName}
// 处理流程：
// 1. 若 service 变化，则判断该 namespace 是否具有 annotation "kuber.io/virtualdomain={virtualdomain name}"
// 2. 判断 service 是否具有annotation "kuber.io/virtualdomain={virtualdomain name}"
// 3. 确定service同名的 serviceentries 是否存在并 update

type ServiceEntryReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch
//+kubebuilder:rbac:groups=networking.istio.io,resources=serviceentries,verbs=*
//+kubebuilder:rbac:groups=networking.istio.io,resources=virtualservices,verbs=*

// TODO
func (r *ServiceEntryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *ServiceEntryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		Watches(&source.Kind{Type: &corev1.Namespace{}}, handler.EnqueueRequestsFromMapFunc(OnNamespaceChangeFunc(mgr.GetClient()))).
		Complete(r)
}

func OnNamespaceChangeFunc(cli client.Client) handler.MapFunc {
	return func(obj client.Object) []reconcile.Request {
		switch data := obj.(type) {
		case *corev1.Namespace:
			ctx := context.Background()

			services := &corev1.ServiceList{}
			if err := cli.List(ctx, services, client.InNamespace(data.Name)); err != nil {
				return []reconcile.Request{}
			}
			requests := make([]reconcile.Request, len(services.Items))
			for i, svc := range services.Items {
				requests[i] = reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&svc)}
			}
			return requests
		default:
			return []reconcile.Request{}
		}
	}
}
