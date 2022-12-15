package indexer

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CustomIndexPods(c cache.Cache) error {
	if err := c.IndexField(context.TODO(), &v1.Pod{}, "phase", func(o client.Object) []string {
		value := o.(*v1.Pod)
		return []string{podStatus(value)}
	}); err != nil {
		return err
	}

	if err := c.IndexField(context.TODO(), &v1.Pod{}, "nodename", func(o client.Object) []string {
		value := o.(*v1.Pod)
		return []string{podStatus(value)}
	}); err != nil {
		return err
	}

	return nil
}

func podStatus(pod *v1.Pod) string {
	if pod.GetDeletionTimestamp() != nil {
		return "Terminating"
	}

	if len(pod.Status.ContainerStatuses) == 0 {
		if len(pod.Status.Reason) > 0 {
			return pod.Status.Reason
		} else {
			return string(pod.Status.Phase)
		}
	}

	st := "Running"

	for _, co := range pod.Status.ContainerStatuses {
		if co.State.Waiting != nil {
			st = co.State.Waiting.Reason
		} else if co.State.Terminated != nil {
			st = co.State.Terminated.Reason
		}
	}
	return st
}
