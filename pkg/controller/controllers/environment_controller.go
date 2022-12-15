package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/sunweiwe/kuber/pkg/api/kuber"
	"github.com/sunweiwe/kuber/pkg/api/kuber/v1beta1"
	"github.com/sunweiwe/kuber/pkg/utils/maps"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type EnvironmentReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=go.kuber.io,resources=environments,verbs=get;list;watch;create;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups=go.kuber.io,resources=environments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=resourcequotas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=limitranges,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watchc
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete

func (r *EnvironmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		环境逻辑：
		1. 创建,更新或者关联Namespace, 给namespace打标签
		2. 创建或者更新ResourceQuota,打标签
		3. 创建或者更新LimitRange,打标签
		4. 创建的时候，添加finalizer;删除得时候，根据策略删除对应的ns,或者删除label
	*/

	log := r.Log.WithName("Environment").WithValues("Environment", req.Name)
	var env v1beta1.Environment
	if err := r.Get(ctx, req.NamespacedName, &env); err != nil {
		log.Info("Failed to get Environment")
		return ctrl.Result{}, nil
	}

	Labels := map[string]string{
		kuber.LabelProject:     env.Spec.Project,
		kuber.LabelTenant:      env.Spec.Tenant,
		kuber.LabelEnvironment: env.Name,
	}

	owner := metav1.NewControllerRef(&env, v1beta1.SchemeEnvironment)

	if !env.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&env, kuber.FinalizerNamespace) {
			if err := r.handleDelete(&env, Labels, owner, ctx, log); err != nil {
				log.Error(err, "Failed to delete Environment related namespace")
				return ctrl.Result{Requeue: true}, err
			}
			controllerutil.RemoveFinalizer(&env, kuber.FinalizerNamespace)
			if err := r.Update(ctx, &env); err != nil {
				log.Error(err, "Failed to delete Environment finalizer")
				return ctrl.Result{Requeue: true}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *EnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.Environment{}).
		Complete(r)
}

func (r *EnvironmentReconciler) handleDelete(env *v1beta1.Environment, Labels map[string]string, owner *metav1.OwnerReference, ctx context.Context, log logr.Logger) error {
	var namespace corev1.Namespace
	namespaceKey := types.NamespacedName{
		Name: env.Spec.Namespace,
	}

	if err := r.Get(ctx, namespaceKey, &namespace); err != nil {
		if i := client.IgnoreNotFound(err); i != nil {
			return i
		}
	}

	if env.Spec.DeletePolicy == v1beta1.DeletePolicyLabels {
		namespace.ObjectMeta.Labels = maps.LabelDelete(namespace.ObjectMeta.Labels, Labels)
		namespace.SetOwnerReferences(nil)
		if err := r.Update(ctx, &namespace); err != nil {
			r.Recorder.Eventf(env, corev1.EventTypeWarning, ReasonFailedDelete, "Failed to delete environment labels for namespace %s", env.Spec.Namespace)
			return err
		}
		r.Recorder.Eventf(env, corev1.EventTypeNormal, ReasonDeleted, "Successfully to delete environment labels for namespace %s", env.Spec.Namespace)
	}

	return nil
}
