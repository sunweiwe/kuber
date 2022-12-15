package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/sunweiwe/kuber/pkg/api/kuber"
	"github.com/sunweiwe/kuber/pkg/api/kuber/v1beta1"
	"github.com/sunweiwe/kuber/pkg/controller/handler"
	corev1 "k8s.io/api/core/v1"
	networkV1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	isoKindTenant     = "tenant"
	isoKindProject    = "project"
	isKindEnvironment = "environment"
)

type NetworkPolicyAction struct {
	TenantISO      bool
	ProjectISO     bool
	EnvironmentISO bool

	Tenant      string
	Project     string
	Environment string

	Origin *networkV1.NetworkPolicy
	Modify *networkV1.NetworkPolicy

	Labels map[string]string

	action string
}

type TenantNetworkPolicyReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=go.kuber.io,resources=tenantnetworkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=go.kuber.io,resources=tenantnetworkpolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=go.kuber.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete

func (r *TenantNetworkPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("tenantNetworkPolicy", req.NamespacedName)
	var networks v1beta1.TenantNetworkPolicy

	if err := r.Get(ctx, req.NamespacedName, &networks); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		} else {
			log.Error(err, "failed to get tenantNetworkPolicy")
		}
	}

	if !networks.ObjectMeta.DeletionTimestamp.IsZero() {
		tenantName, exist := networks.Labels[kuber.LabelTenant]
		if !exist {
			log.Error(
				fmt.Errorf("failed to delete tenantNetworkPolicy [%s] related networkPolicies, tenant label not exist", networks.Name),
				"")
			return ctrl.Result{}, nil
		}

		networkList := networkV1.NetworkPolicyList{}
		r.List(ctx, &networkList, &client.ListOptions{
			Namespace: corev1.NamespaceAll,
			LabelSelector: labels.SelectorFromSet(map[string]string{
				kuber.LabelTenant: tenantName,
			}),
		})

		for _, np := range networkList.Items {
			r.Delete(ctx, &np)
		}
		networks.SetOwnerReferences(nil)
		networks.SetFinalizers(nil)
		r.Update(ctx, &networks)
		return ctrl.Result{}, nil
	}

	statusMap := map[string]NetworkPolicyAction{}

	for _, action := range statusMap {
		switch action.action {
		case "create":
			action.Modify.Labels = labels.Merge(action.Modify.Labels, action.Labels)
			if err := r.Create(ctx, action.Modify); err != nil {
				log.Info("Error create networkPolicy" + err.Error())
			}
		case "delete":
			if err := r.Delete(ctx, action.Modify); err != nil {
				log.Info("Error delete networkPolicy " + err.Error())
			}
		case "update":
			action.Modify.Labels = labels.Merge(action.Modify.Labels, action.Labels)
			if err := r.Update(ctx, action.Modify); err != nil {
				log.Info("Error update networkPolicy " + err.Error())
			}
		default:
			continue

		}
	}

	if !controllerutil.ContainsFinalizer(&networks, kuber.FinalizerNetworkPolicy) {
		controllerutil.AddFinalizer(&networks, kuber.FinalizerNetworkPolicy)
		r.Update(ctx, &networks)
	}

	return ctrl.Result{}, nil
}

func (r *TenantNetworkPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.TenantNetworkPolicy{}).
		Watches(&source.Kind{Type: &corev1.Node{}}, handler.NewNodeHandler(r.Client, r.Log)).
		Complete(r)
}
