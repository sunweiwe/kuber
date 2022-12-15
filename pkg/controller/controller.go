//Package controller manage
package controller

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"github.com/sunweiwe/kuber/pkg/api/kuber"
	"github.com/sunweiwe/kuber/pkg/controller/controllers"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme        = runtime.NewScheme()
	setupLog      = ctrl.Log.WithName("setup")
	leaseDuration = 30 * time.Second
	renewDeadline = 20 * time.Second
)

type Options struct {
	MetricsAddr          string `json:"metricsAddr,omitempty" description:"The address the metric endpoint binds to."`
	ProbeAddr            string `json:"probeAddr,omitempty" description:"The address the probe endpoint binds to."`
	WebhookAddr          string `json:"webhookAddr,omitempty" description:"The address the webhook endpoint binds to."`
	EnableLeaderElection bool   `json:"enableLeaderElection,omitempty" description:"Enable leader election for controller manager."`
	EnableWebhook        bool   `json:"enableWebhook,omitempty" description:"Enable webhook for controller manager."`
	Repository           string `json:"repository,omitempty" description:"default image repo."`
}

func NewDefaultOptions() *Options {
	return &Options{
		WebhookAddr:          ":9443",
		MetricsAddr:          ":9090",
		ProbeAddr:            ":8081",
		EnableLeaderElection: false,
		EnableWebhook:        true,
		Repository:           "docker.io/kuber/ingress-nginx-operator",
	}
}

func Run(ctx context.Context, options *Options) error {
	ctrl.SetLogger(zap.New(zap.UseDevMode(false)))

	webhookHost, webhookPortStr, err := net.SplitHostPort(options.WebhookAddr)
	if err != nil {
		return fmt.Errorf("parse webhook address: %v", err)
	}
	webhookPort, err := strconv.Atoi(webhookPortStr)
	if err != nil {
		return fmt.Errorf("parse webhook port: %v", err)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     options.MetricsAddr,
		Port:                   webhookPort,
		Host:                   webhookHost,
		HealthProbeBindAddress: options.ProbeAddr,
		LeaseDuration:          &leaseDuration,
		RenewDeadline:          &renewDeadline,
		LeaderElection:         options.EnableLeaderElection,
		LeaderElectionID:       kuber.GroupName,
	})

	if err != nil {
		setupLog.Error(err, "unable to create manager")
		return err
	}

	// setup healthz
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		return err
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		return err
	}

	// setup controllers
	if err := setupControllers(mgr, options, setupLog); err != nil {
		return err
	}

	if options.EnableWebhook {

	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		return err
	}
	return nil
}

func setupControllers(mgr ctrl.Manager, options *Options, setupLog logr.Logger) error {
	if err := (&controllers.TenantReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Log:      ctrl.Log.WithName("controllers").WithName("Tenant"),
		Recorder: mgr.GetEventRecorderFor("Tenant"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Tenant")
		return err
	}

	if err := (&controllers.TenantResourceQuotaReconciler{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TenantResourceQuota")
		return err
	}

	if err := (&controllers.TenantNetworkPolicyReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Log:      ctrl.Log.WithName("controllers").WithName("TenantNetworkPolicy"),
		Recorder: mgr.GetEventRecorderFor("TenantNetworkPolicy"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TenantGateway")
		return err
	}

	if err := (&controllers.EnvironmentReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Log:      ctrl.Log.WithName("controllers").WithName("Environment"),
		Recorder: mgr.GetEventRecorderFor("Environment"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Environment")
		return err
	}

	if err := (&controllers.ServiceEntryReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Log:    ctrl.Log.WithName("controllers").WithName("Environment"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Environment")
		return err
	}

	return nil
}

func setupWebhook(mgr ctrl.Manager, setupLog logr.Logger, repository string) error {
	setupLog.Info("registering webhooks")

	// ws := mgr.GetWebhookServer()
	// c := mgr.GetClient()

	// validateLogger := ctrl.Log.WithName("validate-webhook")
	// validateHandler:=webhooks.Getv

	return nil
}
