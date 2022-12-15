package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/sunweiwe/kuber/pkg/api/kuber"
	"github.com/sunweiwe/kuber/pkg/api/kuber/v1beta1"
	"github.com/sunweiwe/kuber/pkg/controller/handler"
	"github.com/sunweiwe/kuber/pkg/utils/maps"
	"github.com/sunweiwe/kuber/pkg/utils/resourcequota"
	"github.com/sunweiwe/kuber/pkg/utils/slice"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type TenantReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=go.kuber.io,resources=tenants,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=go.kuber.io,resources=tenants/status,verbs=get;update;patch

func (r *TenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		租户逻辑:
		1. 创建或者更新TenantResourceQuota
		2. 创建或者更新TenantNetworkPolicy
		3. 创建或者更新TenantGateway
	*/
	log := r.Log.WithName("Tenant").WithValues("tenant", req.Name)
	var tenant v1beta1.Tenant
	if err := r.Get(ctx, req.NamespacedName, &tenant); err != nil {
		if errors.IsNotFound(err) {
			log.Info("NotFound")
			return ctrl.Result{}, nil
		} else {
			log.Info("Failed to get Tenant")
			return ctrl.Result{}, nil
		}
	}

	ref := metav1.NewControllerRef(&tenant, v1beta1.SchemeTenant)

	// 删除操作，前台删除
	if !tenant.ObjectMeta.DeletionTimestamp.IsZero() {

		// 删除所有得环境
		if controllerutil.ContainsFinalizer(&tenant, kuber.FinalizerNetworkPolicy) {
			env := v1beta1.Environment{}
			if err := r.DeleteAllOf(ctx, &env, &client.DeleteAllOfOptions{
				ListOptions: client.ListOptions{
					LabelSelector: labels.SelectorFromSet(map[string]string{
						kuber.LabelTenant: tenant.Spec.TenantName,
					}),
				},
			}); err != nil {
				log.Error(err, "failed to delete environment")
			}
			controllerutil.RemoveFinalizer(&tenant, kuber.FinalizerEnvironment)
		}

		// 删除TenantResourceQuota
		if controllerutil.ContainsFinalizer(&tenant, kuber.FinalizerResourceQuota) {
			trq := &v1beta1.TenantResourceQuota{}
			key := types.NamespacedName{
				Name: tenant.Spec.TenantName,
			}
			controllerutil.RemoveFinalizer(&tenant, kuber.FinalizerResourceQuota)
			if err := r.Get(ctx, key, trq); err != nil {
				if !errors.IsNotFound(err) {
					log.Error(err, "failed to get tenant resource quota")
				}
			} else {
				now := metav1.Now()
				trq.SetOwnerReferences(nil)
				trq.SetDeletionTimestamp(&now)
				r.Update(ctx, trq)
			}
		}

		// 删除TenantResourceQuota
		if controllerutil.ContainsFinalizer(&tenant, kuber.FinalizerResourceQuota) {
			trq := &v1beta1.TenantResourceQuota{}
			key := types.NamespacedName{
				Name: tenant.Spec.TenantName,
			}
			controllerutil.RemoveFinalizer(&tenant, kuber.FinalizerResourceQuota)
			if err := r.Get(ctx, key, trq); err != nil {
				if !errors.IsNotFound(err) {
					log.Error(err, "failed to get tenant resource quota")
				}
			} else {
				now := metav1.Now()
				trq.SetOwnerReferences(nil)
				trq.SetDeletionTimestamp(&now)
				r.Update(ctx, trq)
			}
		}

		// 删除TenantNetworkPolicy
		if controllerutil.ContainsFinalizer(&tenant, kuber.FinalizerNetworkPolicy) {
			t := &v1beta1.TenantNetworkPolicy{}
			key := types.NamespacedName{
				Name: tenant.Spec.TenantName,
			}
			controllerutil.RemoveFinalizer(&tenant, kuber.FinalizerNetworkPolicy)
			if err := r.Get(ctx, key, t); err != nil {
				if !errors.IsNotFound(err) {
					log.Error(err, "failed to get tenant network policy")
				}
			} else {
				now := metav1.Now()
				t.SetOwnerReferences(nil)
				t.SetDeletionTimestamp(&now)
				r.Update(ctx, t)
			}
		}

		// 删除TenantGateway
		if controllerutil.ContainsFinalizer(&tenant, kuber.FinalizerGateway) {
			tg := &v1beta1.TenantGateway{}
			if err := r.DeleteAllOf(ctx, tg, &client.DeleteAllOfOptions{
				ListOptions: client.ListOptions{
					LabelSelector: labels.SelectorFromSet(map[string]string{
						kuber.LabelTenant: tenant.Spec.TenantName,
					}),
				},
			}); err != nil {
				log.Error(err, "failed to delete tenant gateway")
			}
			controllerutil.RemoveFinalizer(&tenant, kuber.FinalizerGateway)
		}

		r.Update(ctx, &tenant)
		return ctrl.Result{}, nil
	}

	var changed bool
	// TenantResourceQuota
	r.handleTenantResourceQuota(&tenant, ref, ctx, log)
	if !controllerutil.ContainsFinalizer(&tenant, kuber.FinalizerResourceQuota) {
		controllerutil.AddFinalizer(&tenant, kuber.FinalizerResourceQuota)
		changed = true
	}

	//  处理NetworkPolicy
	r.handleTenantNetworkPolicy(&tenant, ref, ctx, log)
	if !controllerutil.ContainsFinalizer(&tenant, kuber.FinalizerNetworkPolicy) {
		controllerutil.AddFinalizer(&tenant, kuber.FinalizerNetworkPolicy)
		changed = true
	}

	// 添加环境 finalizer
	if !controllerutil.ContainsFinalizer(&tenant, kuber.FinalizerEnvironment) {
		controllerutil.AddFinalizer(&tenant, kuber.FinalizerEnvironment)
		changed = true
	}

	// Gateway
	if !controllerutil.ContainsFinalizer(&tenant, kuber.FinalizerGateway) {
		controllerutil.AddFinalizer(&tenant, kuber.FinalizerGateway)
		changed = true
	}

	if changed {
		if err := r.Update(ctx, &tenant); err != nil {
			msg := fmt.Sprintf("Failed to update tenant %s: %v", tenant.Name, err)
			log.Info(msg)
			return ctrl.Result{Requeue: true}, nil
		}
	}

	if r.handleTenantStatus(&tenant, ctx, log) {
		if err := r.Status().Update(ctx, &tenant); err != nil {
			msg := fmt.Sprintf("Failed to update tenant %s: %v", tenant.Name, err)
			log.Info(msg)
			return ctrl.Result{Requeue: true}, nil
		}
	}

	return ctrl.Result{}, nil
}

func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.Tenant{}).
		Watches(&source.Kind{Type: &v1beta1.Environment{}}, handler.NewEnvironmentHandler(r.Client, r.Log)).
		Watches(&source.Kind{Type: &v1beta1.TenantResourceQuota{}}, handler.NewTenantResourceQuotaHandler(r.Client, r.Log)).
		Complete(r)
}

func (r *TenantReconciler) handleTenantStatus(tenant *v1beta1.Tenant, ctx context.Context, log logr.Logger) bool {
	var envs v1beta1.EnvironmentList
	if err := r.List(ctx, &envs, client.MatchingLabels{kuber.LabelTenant: tenant.Spec.TenantName}); err != nil {
		r.Log.Error(err, "failed to list environments")
		return false
	}
	var envNames []string
	var namespaces []string
	for _, env := range envs.Items {
		envNames = append(envNames, env.Name)
		namespaces = append(namespaces, env.Spec.Namespace)
	}

	envEqual := slice.StringArrayEqual(tenant.Status.Environments, envNames)
	nsEqual := slice.StringArrayEqual(tenant.Status.Namespaces, namespaces)

	if envEqual && nsEqual {
		return false
	}

	tenant.Status.Environments = envNames
	tenant.Status.Namespaces = namespaces
	tenant.Status.LastUpdateTime = metav1.Now()
	return false
}

func (r *TenantReconciler) handleTenantResourceQuota(tenant *v1beta1.Tenant, owner *metav1.OwnerReference, ctx context.Context, log logr.Logger) {
	var trq v1beta1.TenantResourceQuota
	trqKey := types.NamespacedName{
		Name: tenant.Spec.TenantName,
	}

	Labels := map[string]string{
		kuber.LabelTenant: tenant.Spec.TenantName,
	}

	if err := r.Get(ctx, trqKey, &trq); err != nil {
		if !errors.IsNotFound(err) {
			log.Info("Failed to get TenantResourceQuota")
			return
		}
		log.Info("Not Found TenantResourceQuota, create one")
		trq.Name = tenant.Name
		controllerutil.SetControllerReference(tenant, &trq, r.Scheme)
		trq.Labels = labels.Merge(trq.Labels, Labels)
		trq.Spec.Hard = resourcequota.GetDefaultTenantResourceQuota()
		if err := r.Create(ctx, &trq); err != nil {
			r.Recorder.Eventf(tenant, corev1.EventTypeWarning, ReasonFailedCreateSubResource, "Failed to Create TenantResourceQuota for tenant %s: %v", tenant.Spec.TenantName, err)
			log.Info("Failed to create TenantResourceQuota: " + err.Error())
			return
		}
		r.Recorder.Eventf(tenant, corev1.EventTypeNormal, ReasonCreated, "Successfully create TenantResourceQuota for tenant %s", tenant.Spec.TenantName)
	}
	var changed bool
	if !ExistOwnerRef(trq.ObjectMeta, *owner) {
		controllerutil.SetControllerReference(tenant, &trq, r.Scheme)
		changed = true
	}
	if maps.LabelChanged(trq.Labels, Labels) {
		trq.Labels = labels.Merge(trq.Labels, Labels)
		changed = true
	}

	if changed {
		if err := r.Update(ctx, &trq); err != nil {
			r.Recorder.Eventf(tenant, corev1.EventTypeWarning, ReasonFailedUpdate, "Failed to update TenantResourceQuota for tenant %s", tenant.Spec.TenantName)
			log.Info("Failed to update TenantResourceQuota")
		}
		r.Recorder.Eventf(tenant, corev1.EventTypeNormal, ReasonUpdated, "Successfully update TenantResourceQuota for tenant %s", tenant.Spec.TenantName)
	}
}

func (r *TenantReconciler) handleTenantNetworkPolicy(tenant *v1beta1.Tenant, owner *metav1.OwnerReference, ctx context.Context, log logr.Logger) {
	var networks v1beta1.TenantNetworkPolicy
	networksKey := types.NamespacedName{
		Name: tenant.Name,
	}

	Labels := map[string]string{
		kuber.LabelTenant: tenant.Name,
	}

	if err := r.Get(ctx, networksKey, &networks); err != nil {
		if !errors.IsNotFound(err) {
			log.Info("Failed to get TenantNetworkPolicy")
			return
		}
		log.Info("NotFound TenantNetworkPolicy, create one")
		networks.Name = tenant.Name
		controllerutil.SetControllerReference(tenant, &networks, r.Scheme)
		networks.Labels = labels.Merge(networks.Labels, Labels)
		networks.Spec.Tenant = tenant.Name
		networks.Spec.TenantIsolated = false
		if err := r.Create(ctx, &networks); err != nil {
			r.Recorder.Eventf(tenant, corev1.EventTypeNormal, ReasonFailedCreateSubResource, "Failed to create TenantNetworkPolicy for tenant %s: %v", tenant.Spec.TenantName, err)
			log.Info("Failed to create TenantNetworkPolicy: " + err.Error())
			return
		}
		r.Recorder.Eventf(tenant, corev1.EventTypeNormal, ReasonCreated, "Successfully create TenantNetworkPolicy for tenant %s", tenant.Spec.TenantName)
	}

	var changed bool
	if !ExistOwnerRef(networks.ObjectMeta, *owner) {
		controllerutil.SetControllerReference(tenant, &networks, r.Scheme)
		changed = true
	}
	if maps.LabelChanged(networks.Labels, Labels) {
		networks.Labels = labels.Merge(networks.Labels, Labels)
		changed = true
	}
	if changed {
		if err := r.Update(ctx, &networks); err != nil {
			r.Recorder.Eventf(tenant, corev1.EventTypeWarning, ReasonFailedUpdate, "Failed to update TenantNetworkPolicy for tenant %s", tenant.Spec.TenantName)
			log.Info("Failed to update TenantNetworkPolicy")
		}
		r.Recorder.Eventf(tenant, corev1.EventTypeNormal, ReasonUpdated, "Successfully update TenantNetworkPolicy for tenant %s", tenant.Spec.TenantName)
	}

}
