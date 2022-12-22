package statistics

import (
	"context"
	"strings"

	"github.com/sunweiwe/kuber/pkg/api/kuber/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const ResourceLimitsPrefix = "limits."

type ClusterResourceStatistics struct {
	// 集群资源的总容量，即物理资源总量
	Capacity corev1.ResourceList `json:"capacity"`
	// 集群资源的真实使用量
	Used corev1.ResourceList `json:"used"`
	// 集群下的租户资源分配总量
	TenantAllocated corev1.ResourceList `json:"tenantAllocated"`
}

type ClusterWorkloadStatistics map[string]int

func GetClusterResourceStatistics(ctx context.Context, cli client.Client) ClusterResourceStatistics {
	nodeList := &corev1.NodeList{}
	_ = cli.List(ctx, nodeList)

	capacity := corev1.ResourceList{}
	free := corev1.ResourceList{}
	for _, node := range nodeList.Items {
		ResourceList(capacity, node.Status.Capacity)
		ResourceList(free, node.Status.Allocatable)
	}

	used := capacity.DeepCopy()
	SubResourceList(used, free)

	tenantAllocated, _ := clusterTenantResourceQuotaLimits(ctx, cli)

	if val, ok := tenantAllocated["requests.storage"]; ok {
		tenantAllocated["limits.storage"] = val.DeepCopy()
	}

	tenantAllocated = FilterResourceName(tenantAllocated, func(name corev1.ResourceName) bool {
		return strings.HasPrefix(string(name), ResourceLimitsPrefix)
	})

	capacity = AppendResourceNamePrefix(ResourceLimitsPrefix, capacity)
	used = AppendResourceNamePrefix(ResourceLimitsPrefix, used)

	return ClusterResourceStatistics{
		Capacity:        capacity,
		Used:            used,
		TenantAllocated: tenantAllocated,
	}
}

func FilterResourceName(list corev1.ResourceList, keep func(name corev1.ResourceName) bool) corev1.ResourceList {
	ret := corev1.ResourceList{}
	for k, v := range list {
		if keep(k) {
			ret[k] = v.DeepCopy()
		}
	}
	return ret
}

func AppendResourceNamePrefix(prefix string, list corev1.ResourceList) corev1.ResourceList {
	ret := corev1.ResourceList{}
	for k, v := range list {
		ret[corev1.ResourceName(prefix)+k] = v.DeepCopy()
	}
	return ret
}

func clusterTenantResourceQuotaLimits(ctx context.Context, cli client.Client) (corev1.ResourceList, error) {
	tenantResourceQuotaList := &v1beta1.TenantResourceQuotaList{}
	if err := cli.List(ctx, tenantResourceQuotaList); err != nil {
		return nil, err
	}
	total := corev1.ResourceList{}
	for _, quota := range tenantResourceQuotaList.Items {
		ResourceList(total, quota.Spec.Hard)
	}
	return total, nil
}

func ResourceList(total corev1.ResourceList, add corev1.ResourceList) {
	ResourceListCollect(total, add, func(_ corev1.ResourceName, q1 *resource.Quantity, q2 resource.Quantity) {
		q1.Add(q2)
	})
}

func SubResourceList(total corev1.ResourceList, add corev1.ResourceList) {
	ResourceListCollect(total, add, func(_ corev1.ResourceName, q1 *resource.Quantity, q2 resource.Quantity) {
		q1.Add(q2)
	})
}

type ResourceListCollectFunc func(corev1.ResourceName, *resource.Quantity, resource.Quantity)

func ResourceListCollect(into, values corev1.ResourceList, collect ResourceListCollectFunc) corev1.ResourceList {
	for resourceName, quantity := range values {
		lastQuantity := into[resourceName].DeepCopy()
		collect(resourceName, &lastQuantity, quantity)
		into[resourceName] = lastQuantity
	}

	return into
}
