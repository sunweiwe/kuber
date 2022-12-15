//Package apis for client
package apis

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sunweiwe/kuber/pkg/agent/cluster.go"
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

func Run(ctx context.Context, cluster cluster.Interface, system *system.Options, options *Options, debugOptions *DebugOptions) error {
	G := gin.New()

	G.Use(
		log.DefaultGinLoggerMiddleware(),
		//
		exporter.GetRequestCollector().HandlerFunc(),

		// recovery
		gin.Recovery(),
	)

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

	//TODO

	return nil
}
