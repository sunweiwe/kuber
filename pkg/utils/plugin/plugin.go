package plugin

import (
	"context"

	pluginsCommon "github.com/sunweiwe/kuber/pkg/api/plugin"
	"github.com/sunweiwe/kuber/pkg/api/plugin/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PluginManager struct {
	CacheDir string
	Client   client.Client
}

func (m *PluginManager) ListInstalled(ctx context.Context, checkHealthy bool) (map[string]PluginVersion, error) {
	pluginList := &v1beta1.PluginList{}
	if err := m.Client.List(ctx, pluginList, client.InNamespace(pluginsCommon.KuberNamespaceInstaller)); err != nil {
		return nil, err
	}

	ret := map[string]PluginVersion{}
	for _, plugin := range pluginList.Items {
		pv := PluginVersionFrom(plugin)
		if checkHealthy {

		}
		ret[plugin.Name] = pv

	}

	return ret, nil
}
