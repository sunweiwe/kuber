package webhooks

import (
	"context"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type ResourceValidate struct {
	Client  client.Client
	decoder *admission.Decoder
	Log     logr.Logger
}

func GetValidateHandler(client *client.Client, log *logr.Logger) *webhook.Admission {
	return &webhook.Admission{Handler: &ResourceValidate{Client: *client, Log: *log}}
}

func (r *ResourceValidate) Handle(ctx context.Context, req admission.Request) admission.Response {
	switch req.Kind {
	// case gkvTenant:
	// 	return r.ValidateTenant(ctx, req)
	// case gkvTenantResourceQuota:
	// 	return r.ValidateTenantResourceQuota(ctx, req)
	// case gkvTenantGateway:
	// 	return r.ValidateTenantGateway(ctx, req)
	// case gkvTenantNetworkPolicy:
	// 	return r.ValidateTenantNetworkPolicy(ctx, req)
	// case gkvEnvironment:
	// 	return r.ValidateEnvironment(ctx, req)
	// case gkvNamespace:
	// 	return r.ValidateNamespace(ctx, req)
	// case gvkIstioGateway:
	// 	return r.ValidateIstioGateway(ctx, req)
	default:
		return admission.Allowed("pass")
	}
}
