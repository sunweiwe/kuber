package exporter

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sunweiwe/kuber/pkg/agent/cluster.go"
	"github.com/sunweiwe/kuber/pkg/log"
	"github.com/sunweiwe/kuber/pkg/utils/plugin"
)

type PluginCollector struct {
	pluginStatus *prometheus.Desc
	cluster      cluster.Interface
	mutex        sync.Mutex
}

func NewPluginCollectorFunc(cluster cluster.Interface) func(*log.Logger) (Collector, error) {
	return func(l *log.Logger) (Collector, error) {
		return NewPluginCollector(l, cluster)
	}

}

func NewPluginCollector(_ *log.Logger, cluster cluster.Interface) (Collector, error) {
	c := &PluginCollector{
		pluginStatus: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace(), "plugin", "status"), "Kuber plugin status",
			[]string{"type", "plugin", "namespace", "enabled", "version"}, nil),
		cluster: cluster,
	}
	return c, nil
}

func (c *PluginCollector) Update(ch chan<- prometheus.Metric) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	pm := plugin.PluginManager{Client: c.cluster.GetClient()}
	plugins, err := pm.ListInstalled(ctx, true)
	if err != nil {
		log.Error(err, "get plugins failed")
		return err
	}

	for _, p := range plugins {
		ch <- prometheus.MustNewConstMetric(c.pluginStatus, prometheus.GaugeValue,
			func() float64 {
				if p.Healthy {
					return 1
				}
				return 0
			}(),
			p.MainCategory, p.Name, p.Namespace, strconv.FormatBool(p.Enabled), p.Version,
		)
	}
	return nil
}
