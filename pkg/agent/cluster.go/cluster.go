package cluster

import (
	"context"
	"strings"

	"github.com/sunweiwe/kuber/pkg/kube"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsV1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
)

type Interface interface {
	cluster.Cluster
	Config() *rest.Config
	Kubernetes() kubernetes.Interface
	Discovery() discovery.CachedDiscoveryInterface
	Watch(ctx context.Context, list client.ObjectList, callback func(watch.Event) error, opts ...client.ListOption) error
}

type Cluster struct {
	cluster.Cluster
	config     *rest.Config
	discovery  discovery.CachedDiscoveryInterface
	kubernetes kubernetes.Interface
}

func WithDefaultScheme(o *cluster.Options) {
	o.Scheme = kube.Scheme()
}

func NewCluster(config *rest.Config, options ...cluster.Option) (*Cluster, error) {
	discovery, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}

	options = append(options, WithDefaultScheme)
	kubernetesClientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	c, err := cluster.New(config, options...)
	if err != nil {
		return nil, err
	}

	return &Cluster{
		Cluster:    c,
		kubernetes: kubernetesClientSet,
		config:     config,
		discovery:  memory.NewMemCacheClient(discovery),
	}, nil
}

func WithDisableCaches() func(o *cluster.Options) {
	disabled := []client.Object{
		&metricsV1beta1.NodeMetrics{},
		&metricsV1beta1.PodMetrics{},
	}

	return func(o *cluster.Options) { o.ClientDisableCacheFor = append(o.ClientDisableCacheFor, disabled...) }
}

func (c *Cluster) Kubernetes() kubernetes.Interface {
	return c.kubernetes
}

func (c *Cluster) Config() *rest.Config {
	return c.config
}

func (c *Cluster) Discovery() discovery.CachedDiscoveryInterface {
	return c.discovery
}

func (c *Cluster) Watch(ctx context.Context, list client.ObjectList, callback func(watch.Event) error, opts ...client.ListOption) error {
	gvk, err := apiutil.GVKForObject(list, c.GetScheme())
	if err != nil {
		return err
	}
	gvk.Kind = strings.TrimSuffix(gvk.Kind, "List")

	mapping, err := c.Cluster.GetRESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	if callback == nil {
		return errors.NewBadRequest("no callback provided")
	}

	listOpts := client.ListOptions{}
	listOpts.ApplyOptions(opts)

	config := c.config
	dy, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}

	watcher, err := dy.Resource(mapping.Resource).Namespace(listOpts.Namespace).Watch(ctx, *listOpts.AsListOptions())
	if err != nil {
		return err
	}
	defer watcher.Stop()

	for {
		select {
		case event := <-watcher.ResultChan():
			if err := callback(event); err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}

}
