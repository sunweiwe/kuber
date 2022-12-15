package plugin

import (
	"encoding/json"

	pluginsCommon "github.com/sunweiwe/kuber/pkg/api/plugin"
	"github.com/sunweiwe/kuber/pkg/api/plugin/v1beta1"
)

type PluginVersion struct {
	Name             string               `json:"name,omitempty"`
	Namespace        string               `json:"namespace,omitempty"`
	Enabled          bool                 `json:"enabled,omitempty"`
	InstallNamespace string               `json:"installNamespace,omitempty"`
	Kind             v1beta1.BundleKind   `json:"kind,omitempty"`
	Description      string               `json:"description,omitempty"`
	HealthCheck      string               `json:"healthCheck,omitempty"`
	MainCategory     string               `json:"mainCategory,omitempty"`
	Category         string               `json:"category,omitempty"`
	Repository       string               `json:"repository,omitempty"`
	Version          string               `json:"version,omitempty"`
	Healthy          bool                 `json:"healthy,omitempty"`
	Required         bool                 `json:"required,omitempty"`
	Requirements     string               `json:"requirements,omitempty"` // dependecies requirements
	Message          string               `json:"message,omitempty"`
	Values           v1beta1.Values       `json:"values,omitempty"`
	Schema           string               `json:"schema"`
	ValuesFrom       []v1beta1.ValuesFrom `json:"valuesFrom,omitempty"`
}

func PluginVersionFrom(plugin v1beta1.Plugin) PluginVersion {
	annotations := plugin.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}
	pv := PluginVersion{}
	_ = json.Unmarshal([]byte(annotations[pluginsCommon.AnnotationIsPlugin]), &pv)

	pv.Name = plugin.Name
	pv.Namespace = plugin.Namespace
	pv.InstallNamespace = plugin.Spec.InstallNamespace
	pv.Version = plugin.Spec.Version
	pv.Enabled = plugin.DeletionTimestamp == nil && !plugin.Spec.Disabled
	pv.Repository = plugin.Spec.URL
	pv.Message = plugin.Status.Message
	if pv.Version == "" {
		pv.Version = plugin.Status.Version
	}
	pv.ValuesFrom = plugin.Spec.ValuesFrom
	if plugin.Status.Phase == v1beta1.PhaseInstalled {
		pv.Healthy = true
	}
	pv.Values = plugin.Spec.Values
	if pv.Description == "" {
		pv.Description = annotations[pluginsCommon.AnnotationPluginDescription]
	}
	category(&pv, annotations)
	return pv
}

// TODO
func category(pv *PluginVersion, annotations map[string]string) {
	full := annotations[pluginsCommon.AnnotationCategory]
	if full == "" {
		return
	}

}
