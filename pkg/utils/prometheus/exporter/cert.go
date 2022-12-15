package exporter

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sunweiwe/kuber/pkg/log"
	"github.com/sunweiwe/kuber/pkg/utils/cluster"
)

type CertCollector struct {
	certExpiredAt *prometheus.Desc
	mutex         sync.Mutex
}

func NewCertCollectorFunc() func(*log.Logger) (Collector, error) {
	return func(l *log.Logger) (Collector, error) {
		return NewCertCollector(l)
	}
}

func NewCertCollector(_ *log.Logger) (Collector, error) {
	c := &CertCollector{
		certExpiredAt: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace(), "cluster_component_cert", "expiration_remain_seconds"),
			"Kuber cluster component cert expiration remain seconds",
			[]string{"component"},
			nil),
	}
	return c, nil
}

func (c *CertCollector) Update(ch chan<- prometheus.Metric) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	expiredAt, err := cluster.GetServerCertExpiredTime(cluster.APIServerURL)
	if err != nil {
		return err
	}
	ch <- prometheus.MustNewConstMetric(
		c.certExpiredAt,
		prometheus.GaugeValue,
		time.Until(*expiredAt).Seconds(),
		"apiserver",
	)

	return nil
}
