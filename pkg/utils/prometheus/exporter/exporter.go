package exporter

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"

	"github.com/prometheus/client_golang/prometheus"
	promCollectors "github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/sunweiwe/kuber/pkg/log"
	"github.com/sunweiwe/kuber/pkg/utils"

	prometheusKuber "github.com/sunweiwe/kuber/pkg/utils/prometheus"

	"go.uber.org/zap"
)

var (
	MetricPath             = utils.StrOrDef(os.Getenv("METRIC_PATH"), "/metrics")
	IncludeExporterMetrics = false
	MaxRequests            = 40
)

type Handler struct {
	unfilteredHandler http.Handler

	exporterMetricsRegistry *prometheus.Registry
	includeExporterMetrics  bool
	maxRequests             int
	logger                  *log.Logger
}

func NewHandler(namespace string, collectors map[string]CollectorFunc) *Handler {
	setNamespace(namespace)
	for k, v := range collectors {
		registerCollector(k, v)
	}

	return newHandler(IncludeExporterMetrics, MaxRequests, log.GlobalLogger.Sugar())
}

func (h *Handler) Run(ctx context.Context, opts *prometheusKuber.ExporterOptions) error {
	mu := http.NewServeMux()
	mu.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html>
			<head><title>Kuber Exporter</title></head>
			<body>
			<h1>Kuber Exporter</h1>
			<p><a href="` + MetricPath + `">Metrics</a></p>
			</body>
			</html>`))
	})
	mu.Handle(MetricPath, h)

	server := &http.Server{Addr: opts.Listen, Handler: mu}
	go func() {
		<-ctx.Done()
		server.Close()
		log.Info("prometheus exporter stopped")
	}()
	log.FromContextOrDiscard(ctx).Info("prometheus exporter listen on", "address", opts.Listen)
	return server.ListenAndServe()
}

func newHandler(includeExporterMetrics bool, maxRequest int, logger *log.Logger) *Handler {
	h := &Handler{
		exporterMetricsRegistry: prometheus.NewRegistry(),
		includeExporterMetrics:  includeExporterMetrics,
		maxRequests:             maxRequest,
		logger:                  logger,
	}

	if h.includeExporterMetrics {
		h.exporterMetricsRegistry.MustRegister(
			promCollectors.NewProcessCollector(promCollectors.ProcessCollectorOpts{}),
			promCollectors.NewGoCollector(),
		)
	}

	if innerHandler, err := h.innerHandler(); err != nil {
		panic(fmt.Sprintf("Couldn't create metrics handler: %s", err))
	} else {
		h.unfilteredHandler = innerHandler
	}

	return h
}

func (h *Handler) innerHandler(filters ...string) (http.Handler, error) {
	ns, err := newBaseCollector(h.logger, filters...)
	if err != nil {
		return nil, fmt.Errorf("couldn't create collector: %s", err)
	}

	if len(filters) == 0 {
		h.logger.Info("Enabled collectors")
		collectors := []string{}
		for n := range ns.Collectors {
			collectors = append(collectors, n)
		}
		sort.Strings(collectors)
		for _, c := range collectors {
			h.logger.Info("collector ", c)
		}
	}

	r := prometheus.NewRegistry()
	r.MustRegister(version.NewCollector(Namespace()))

	if err := r.Register(ns); err != nil {
		return nil, fmt.Errorf("couldn't register node collector: &s", err)
	}

	handler := promhttp.HandlerFor(

		prometheus.Gatherers{h.exporterMetricsRegistry, r},
		promhttp.HandlerOpts{
			ErrorLog:            zap.NewStdLog(h.logger.Desugar()),
			ErrorHandling:       promhttp.ContinueOnError,
			MaxRequestsInFlight: h.maxRequests,
			Registry:            h.exporterMetricsRegistry,
		},
	)

	if h.includeExporterMetrics {
		handler = promhttp.InstrumentMetricHandler(
			h.exporterMetricsRegistry, handler,
		)
	}
	return handler, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	filters := r.URL.Query()["collect[]"]
	h.logger.Debug("filters ", filters)

	if len(filters) == 0 {
		// No filters, use the prepared unfiltered handler.
		h.unfilteredHandler.ServeHTTP(w, r)
		return
	}

	filteredHandler, err := h.innerHandler(filters...)
	if err != nil {
		h.logger.Warn("Couldn't create filtered metrics handler: ", err)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Couldn't create filtered metrics handler: %s", err)
		return
	}
	filteredHandler.ServeHTTP(w, r)
}
