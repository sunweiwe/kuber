package plugin

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	v1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CheckHealthy(ctx context.Context, client client.Client, plugin *PluginVersion) {
	if !plugin.Enabled {
		plugin.Healthy = false
		return
	}

	messages := []string{}
	for _, checkExpression := range strings.Split(plugin.HealthCheck, ",") {
		splits := strings.Split(checkExpression, "/")
		const lenResourceAndName = 2
		if len(splits) != lenResourceAndName {
			continue
		}
		resource, nameRegexp := splits[0], splits[1]
		if err := checkHealthItem(ctx, client, resource, plugin.Namespace, nameRegexp); err != nil {
			messages = append(messages, err.Error())
		}
	}

	if len(messages) > 0 {
		plugin.Message = strings.Join(messages, ",")
		plugin.Healthy = false
	} else {
		plugin.Healthy = true
	}
}

// TODO
func checkHealthItem(ctx context.Context, cli client.Client, resource, namespace, nameRegexp string) error {
	switch {
	case strings.Contains(strings.ToLower(resource), "deployment"):
		deploymentList := &v1.DeploymentList{}
		_ = cli.List(ctx, deploymentList, client.InNamespace(namespace))
		return matchAndCheck(deploymentList.Items, nameRegexp, func(dep v1.Deployment) error {
			if dep.Status.ReadyReplicas != dep.Status.Replicas {
				return fmt.Errorf("Deployment %s is not ready", dep.Name)
			}
			return nil
		})
	}

	return nil
}

func matchAndCheck[T any](list []T, exp string, check func(T) error) error {
	var messages []string
	for _, item := range list {
		value, ok := any(item).(client.Object)
		if !ok {
			value, ok = any(&item).(client.Object)
		}
		if !ok {
			continue
		}
		match, _ := regexp.MatchString(exp, value.GetName())
		if !match {
			continue
		}
		if err := check(item); err != nil {
			messages = append(messages, err.Error())
		}
	}
	if len(messages) > 0 {
		return fmt.Errorf("%s", strings.Join(messages, ","))
	}
	return nil
}
