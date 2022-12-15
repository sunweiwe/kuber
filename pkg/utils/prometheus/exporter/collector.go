package exporter

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sunweiwe/kuber/pkg/log"
)

var (
	// Namespace defines the common namespace to be used by all metrics.
	namespace = "kuber_server"

	scrapeDurationDesc *prometheus.Desc
	scrapeSuccessDesc  *prometheus.Desc
)

func Namespace() string {
	return namespace
}

func setNamespace(ns string) {
	namespace = ns
	scrapeSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_success"),
		"node_exporter: Whether a collector succeeded",
		[]string{"collector"},
		nil,
	)
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_duration_seconds"),
		"node_exporter: Duration of a collector scrape.",
		[]string{"collector"},
		nil,
	)
}

type CollectorFunc func(logger *log.Logger) (Collector, error)

type Collector interface {
	Update(ch chan<- prometheus.Metric) error
}

func registerCollector(collector string, factory CollectorFunc) {
	flag := true
	collectorState[collector] = &flag
	factories[collector] = factory
}

var (
	factories              = make(map[string]CollectorFunc)
	collectorState         = make(map[string]*bool)
	initiatedCollectorsMtx = sync.Mutex{}
	initiatedCollectors    = make(map[string]Collector)
)

type BaseCollector struct {
	Collectors map[string]Collector
	logger     *log.Logger
}

func InitiatedCollectors() map[string]Collector {
	return initiatedCollectors
}

func newBaseCollector(logger *log.Logger, filters ...string) (*BaseCollector, error) {
	f := make(map[string]bool)
	for _, filter := range filters {
		enabled, exist := collectorState[filter]
		if !exist {
			return nil, fmt.Errorf("missing collector: %s", filter)
		}

		if !*enabled {
			return nil, fmt.Errorf("disabled collector: %s", filter)
		}
		f[filter] = true
	}
	collectors := make(map[string]Collector)
	initiatedCollectorsMtx.Lock()
	defer initiatedCollectorsMtx.Unlock()
	for key, enabled := range collectorState {
		if !*enabled || (len(f) > 0 && !f[key]) {
			continue
		}
		if collector, ok := initiatedCollectors[key]; ok {
			collectors[key] = collector
		} else {
			collector, err := factories[key](logger.With("collector", key))
			if err != nil {
				return nil, err
			}
			collectors[key] = collector
			initiatedCollectors[key] = collector
		}
	}
	return &BaseCollector{Collectors: collectors, logger: logger}, nil
}

func (n BaseCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc
}

func (n BaseCollector) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}
	wg.Add(len(n.Collectors))
	for name, c := range n.Collectors {
		go func(name string, c Collector) {
			execute(name, c, ch, n.logger)
			wg.Done()
		}(name, c)
	}
	wg.Wait()
}

func execute(name string, c Collector, ch chan<- prometheus.Metric, logger *log.Logger) {
	begin := time.Now()
	err := c.Update(ch)
	duration := time.Since(begin)
	var success float64

	if err != nil {
		if IsNoDataError(err) {
			logger.Debugf("collector returned no data, name: %s, duration_seconds: %f, err: %v", name, duration.Seconds(), err)
		} else {
			logger.Errorf("collector failed, name: %s, duration_seconds %f, err: %v", name, duration.Seconds(), err)
		}
		success = 0
	} else {
		logger.Debugf("collector succeeded, name: %s, duration_seconds %f", name, duration.Seconds())
		success = 1
	}
	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), name)
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success, name)
}

var ErrNoData = errors.New("collector returned no data")

func IsNoDataError(err error) bool {
	return err == ErrNoData
}
