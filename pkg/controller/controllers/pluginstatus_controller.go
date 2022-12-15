package controllers

import (
	"context"

	"github.com/go-logr/logr"
	apiExtensionsV1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var PluginStatusFactory = &PluginStatus{init: make(chan struct{})}

const (
	ComponentNginx string = "nginx"
	ComponentIstio string = "istio"
)

type PluginStatus struct {
	init                          chan struct{}
	istioOperatorEnabled          bool
	nginxIngressControllerEnabled bool
}

func (p *PluginStatus) ComponentEnabled(name string) bool {
	<-p.init
	switch name {
	case ComponentNginx:
		return p.nginxIngressControllerEnabled
	case ComponentIstio:
		return p.istioOperatorEnabled
	default:
		return false
	}
}

// PluginStatusController 通过crd是否存在以判断对应组件是否被正常安装
// 用于解决当集群中未安装对应crd时，controller 执行产生错误
// 举例： 当istio未安装时，查询istio serviceEntry 会产生错误

//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch

type PluginStatusController struct {
	client.Client
	Log logr.Logger
}

func (r *PluginStatusController) Init(ctx context.Context) error {
	crds := &apiExtensionsV1.CustomResourceDefinitionList{}
	if err := r.Client.List(ctx, crds); err != nil {
		return err
	}

	for _, crd := range crds.Items {
		r.OnChange(ctx, &crd, true)
	}

	close(PluginStatusFactory.init)
	return nil
}

// TODO
func (r *PluginStatusController) OnChange(ctx context.Context, crd *apiExtensionsV1.CustomResourceDefinition, exist bool) (ctrl.Result, error) {

	return ctrl.Result{}, nil
}
