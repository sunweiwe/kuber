//Package apis for client
package apis

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sunweiwe/kuber/pkg/agent/client"
	"github.com/sunweiwe/kuber/pkg/agent/cluster"
	"github.com/sunweiwe/kuber/pkg/agent/middleware"
	"github.com/sunweiwe/kuber/pkg/api/kuber"
	"github.com/sunweiwe/kuber/pkg/log"
	"github.com/sunweiwe/kuber/pkg/utils/prometheus/exporter"
	"github.com/sunweiwe/kuber/pkg/utils/route"
	"github.com/sunweiwe/kuber/pkg/utils/system"
	"github.com/sunweiwe/kuber/pkg/version"
	"k8s.io/apimachinery/pkg/labels"
)

type DebugOptions struct {
	Image       string `json:"image,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	PodSelector string `json:"podSelector,omitempty"`
	Container   string `json:"container,omitempty"`
}

func NewDefaultDebugOptions() *DebugOptions {
	return &DebugOptions{
		Namespace: os.Getenv("KUBER_NAMESPACE"),
		PodSelector: labels.SelectorFromSet(
			labels.Set{
				"app.kubernetes.io/name": "kuber-agent-kubectl",
			}).String(),
		Container: "kuber-agent-kubectl",
		Image:     "kuber/debug-tools:latest",
	}
}

type Options struct {
	PrometheusServer   string `json:"prometheusServer,omitempty"`
	AlertManagerServer string `json:"alertManagerServer,omitempty"`
	LokiServer         string `json:"lokiServer,omitempty"`
	JaegerServer       string `json:"jaegerServer,omitempty"`
	EnableHTTPSigs     bool   `json:"enableHTTPSigs,omitempty" description:"check http sigs, default false"`
}

func NewDefaultOptions() *Options {
	return &Options{
		PrometheusServer:   fmt.Sprintf("http://prometheus.%s:9090", kuber.NamespaceMonitor),
		AlertManagerServer: fmt.Sprintf("http://alertmanager.%s:9090", kuber.NamespaceMonitor),
		LokiServer:         fmt.Sprintf("http://loki-gateway.%s:3100", kuber.NamespaceLogging),
		JaegerServer:       "http://jaeger-query.observability:16686",
		EnableHTTPSigs:     false,
	}
}

type handlerMux struct{ r *route.Router }

const (
	ActionCreate  = "create"
	ActionDelete  = "delete"
	ActionUpdate  = "update"
	ActionPatch   = "patch"
	ActionList    = "list"
	ActionGet     = "get"
	ActionCheck   = "check"
	ActionEnable  = "enable"
	ActionDisable = "disable"
)

// register
func (mu handlerMux) register(group, version, resource, action string, handler gin.HandlerFunc, method ...string) {
	switch action {
	case ActionGet:
		mu.r.MustRegister(http.MethodGet, fmt.Sprintf("/custom/%s/%s/%s/{name}", group, version, resource), handler)
		mu.r.MustRegister(http.MethodGet, fmt.Sprintf("/custom/%s/%s/namespaces/{namespace}/%s/{name}", group, version, resource), handler)
	case ActionList:
		mu.r.MustRegister(http.MethodGet, fmt.Sprintf("/custom/%s/%s/%s", group, version, resource), handler)
		mu.r.MustRegister(http.MethodGet, fmt.Sprintf("/custom/%s/%s/namespaces/{namespace}/%s", group, version, resource), handler)
	default:
		mu.r.MustRegister("*", fmt.Sprintf("/custom/%s/%s/%s/{name}/actions/%s", group, version, resource, action), handler)
		mu.r.MustRegister("*", fmt.Sprintf("/custom/%s/%s/namespaces/{namespace}/%s/{name}/actions/%s", group, version, resource, action), handler)
	}
}

// TODO
func Run(ctx context.Context, cluster cluster.Interface, system *system.Options, options *Options, debugOptions *DebugOptions) error {
	G := gin.New()

	G.Use(
		log.DefaultGinLoggerMiddleware(),
		//
		exporter.GetRequestCollector().HandlerFunc(),

		// recovery
		gin.Recovery(),
	)

	if options.EnableHTTPSigs {
		G.Use(middleware.SignerMiddleware())
	}

	router := route.NewRouter()

	G.Any("/*path", func(ctx *gin.Context) {
		router.Match(ctx)(ctx)
	})

	routes := handlerMux{r: router}
	routes.r.GET("/healthz", func(ctx *gin.Context) {
		content, err := cluster.Kubernetes().Discovery().RESTClient().Get().AbsPath("/healthz").DoRaw(ctx)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"healthy": "notOk",
				"reason":  err.Error(),
			})
			return
		}
		contentStr := string(content)
		if contentStr != "ok" {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"healthy": "notOk",
				"reason":  contentStr,
			})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"healthy": "ok"})
	})

	routes.r.GET("/version", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, version.Get())
	})

	serviceProxyHandler := ServiceProxyHandler{}
	routes.r.ANY("/v1/service-proxy/{realpath}*", serviceProxyHandler.ServiceProxy)

	// restful api for all k8s resources
	routes.registerREST(cluster)

	// custom api
	staticsHandler := &StatisticsHandler{C: cluster.GetClient()}
	routes.register("statistics.system", "v1", "workloads", ActionList, staticsHandler.ClusterWorkloadStatistics)
	routes.register("statistics.system", "v1", "resources", ActionList, staticsHandler.ClusterResourceStatistics)

	nodeHandler := &NodeHandler{C: cluster.GetClient()}
	routes.register("core", "v1", "nodes", ActionGet, nodeHandler.GET)
	routes.register("core", "v1", "nodes", "metadata", nodeHandler.PatchNodeLabelOrAnnotations)
	routes.register("core", "v1", "nodes", "taint", nodeHandler.PatchNodeTaint)
	routes.register("core", "v1", "nodes", "cordon", nodeHandler.PatchNodeCordon)

	nsHandler := &NamespaceHandler{C: cluster.GetClient()}
	routes.register("core", "v1", "namespaces", ActionList, nsHandler.List)

	podHandler := PodHandler{cluster: cluster, debugoptions: debugOptions}
	routes.register("core", "v1", "pods", ActionList, podHandler.List)
	routes.register("core", "v1", "pods", "shell", podHandler.Exec)
	routes.register("core", "v1", "pods", "logs", podHandler.ContainerLogs)
	routes.register("core", "v1", "pods", "file", podHandler.DownloadFile)
	routes.register("core", "v1", "pods", "upfile", podHandler.UploadFile)

	rolloutHandler := &RolloutHandler{cluster: cluster}
	routes.register("apps", "v1", "daemonsets", "rollouthistory", rolloutHandler.DaemonSetHistory)
	routes.register("apps", "v1", "statefulsets", "rollouthistory", rolloutHandler.StatefulSetHistory)
	routes.register("apps", "v1", "deployments", "rollouthistory", rolloutHandler.DeploymentHistory)
	routes.register("apps", "v1", "daemonsets", "rollback", rolloutHandler.DaemonSetRollback)
	routes.register("apps", "v1", "statefulsets", "rollback", rolloutHandler.StatefulSetRollback)
	routes.register("apps", "v1", "deployments", "rollback", rolloutHandler.DeploymentRollback)

	kubectlHandler := KubectlHandler{cluster: cluster, debugOptions: debugOptions}
	routes.register("system", "v1", "kubectl", ActionList, kubectlHandler.Exec)

	prometheusHandler, err := NewPrometheusHandler(options.PrometheusServer)
	if err != nil {
		return err
	}
	routes.register("prometheus", "v1", "vector", ActionList, prometheusHandler.Vector)

	alertManagerHandler, err := NewAlertManagerClient(options.AlertManagerServer, cluster.Kubernetes())
	if err != nil {
		return err
	}
	routes.register("alertmanager", "v1", "alerts", ActionList, alertManagerHandler.ListAlerts)

	jobHandle := &JobHandler{C: cluster.GetClient(), cluster: cluster}
	routes.register("batch", "v1", "jobs", ActionList, jobHandle.List)

	eventHandler := EventHandler{C: cluster.GetClient()}
	routes.register("core", "v1", "events", ActionList, eventHandler.List)

	// service client internal apis
	internalClientRest := client.ClientRest{Cli: cluster.GetClient()}
	internalClientRest.Register(routes.r)

	if err := listen(ctx, system, G); err != nil {
		return err
	}
	return nil
}

func (mu handlerMux) registerREST(cluster cluster.Interface) {
	restHandler := REST{
		client:  cluster.GetClient(),
		cluster: cluster,
	}

	mu.r.GET("/v1/{group}/{version}/{resource}", restHandler.List)
	mu.r.GET("/v1/{group}/{version}/namespaces/{namespace}/{resource}", restHandler.List)

	mu.r.GET("/v1/{group}/{version}/{resource}/{name}", restHandler.Get)
	mu.r.GET("/v1/{group}/{version}/namespaces/{namespace}/{resource}/{name}", restHandler.Get)

	mu.r.POST("/v1/{group}/{version}/{resource}/{name}", restHandler.Create)
	mu.r.POST("/v1/{group}/{version}/namespaces/{namespace}/{resource}/{name}", restHandler.Create)

	mu.r.PUT("/v1/{group}/{version}/{resource}/{name}", restHandler.Update)
	mu.r.PUT("/v1/{group}/{version}/namespaces/{namespace}/{resource}/{name}", restHandler.Update)

	mu.r.PATCH("/v1/{group}/{version}/{resource}/{name}", restHandler.Patch)
	mu.r.PATCH("/v1/{group}/{version}/namespaces/{namespace}/{resource}/{name}", restHandler.Patch)

	mu.r.DELETE("/v1/{group}/{version}/{resource}/{name}", restHandler.Delete)
	mu.r.DELETE("/v1/{group}/{version}/namespaces/{namespace}/{resource}/{name}", restHandler.Delete)

	mu.r.PATCH("/v1/{group}/{version}/{resource}/{name}/actions/scale", restHandler.Scale)
	mu.r.PATCH("/v1/{group}/{version}/namespaces/{namespace}/{resource}/{name}/actions/scale", restHandler.Scale)
}

func listen(ctx context.Context, options *system.Options, handler http.Handler) error {
	server := http.Server{
		BaseContext: func(net.Listener) context.Context { return ctx },
		Addr:        options.Listen,
		Handler:     handler,
	}

	if options.TLSConfigEnabled() {
		tls, err := options.TLSConfig()
		if err != nil {
			return err
		}
		server.TLSConfig = tls
		// server.TLSConfig.ClientAuth = tls.RequireAndVerifyClientCert // enable TLS client auth
	} else {
		log.Info("Tls config not found")
	}

	go func() {
		<-ctx.Done()
		log.Info("Shutting down server")
		server.Close()
	}()

	if server.TLSConfig != nil {
		log.Info("Listen on https", "addr", options.Listen)
		return server.ListenAndServeTLS("", "")
	} else {
		log.Info("Listen on http", "addr", options.Listen)
		return server.ListenAndServe()
	}
}
