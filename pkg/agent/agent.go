//Package agent for client
package agent

import (
	"context"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sunweiwe/kuber/pkg/agent/apis"
	"github.com/sunweiwe/kuber/pkg/agent/cluster"
	"github.com/sunweiwe/kuber/pkg/agent/indexer"
	"github.com/sunweiwe/kuber/pkg/kube"
	"github.com/sunweiwe/kuber/pkg/log"
	"github.com/sunweiwe/kuber/pkg/utils/pprof"
	"github.com/sunweiwe/kuber/pkg/utils/prometheus"
	"github.com/sunweiwe/kuber/pkg/utils/prometheus/exporter"
	"github.com/sunweiwe/kuber/pkg/utils/system"

	"golang.org/x/sync/errgroup"
)

type Options struct {
	DebugMode bool                        `json:"debugMode,omitempty" description:"enable debug mode"`
	LogLevel  string                      `json:"logLevel,omitempty"`
	System    *system.Options             `json:"system,omitempty"`
	API       *apis.Options               `json:"api,omitempty"`
	Debug     *apis.DebugOptions          `json:"debug,omitempty" description:"debug options"`
	Exporter  *prometheus.ExporterOptions `json:"exporter,omitempty"`
}

func NewDefaultOptions() *Options {
	debugMode, _ := strconv.ParseBool(os.Getenv("DEBUG"))
	defaultOptions := &Options{
		DebugMode: debugMode,
		LogLevel:  "debug",
		System:    system.NewDefaultOptions(),
		API:       apis.NewDefaultOptions(),
		Debug:     apis.NewDefaultDebugOptions(),
		Exporter:  prometheus.DefaultExporterOptions(),
	}

	defaultOptions.System.Listen = ":8041"
	return defaultOptions
}

// TODO
func Run(ctx context.Context, options *Options) error {
	log.SetLevel(options.LogLevel)

	if options.DebugMode {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	rest, err := kube.AutoClientConfig()
	if err != nil {
		return err
	}

	c, err := cluster.NewCluster(rest)
	if err != nil {
		return err
	}

	if err := indexer.CustomIndexPods(c.GetCache()); err != nil {
		return err
	}

	go c.Start(ctx)
	c.GetCache().WaitForCacheSync(ctx)

	exporterHandler := exporter.NewHandler("kuber_agent", map[string]exporter.CollectorFunc{
		"plugin":                 exporter.NewPluginCollectorFunc(c), // plugin exporter
		"request":                exporter.NewRequestCollector(),     // http exporter
		"cluster_component_cert": exporter.NewCertCollectorFunc(),    // cluster component cert
	})

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return apis.Run(ctx, c, options.System, options.API, options.Debug)
	})

	eg.Go(func() error {
		return pprof.Run(ctx)
	})

	eg.Go(func() error {
		return exporterHandler.Run(ctx, options.Exporter)
	})

	return nil
}
