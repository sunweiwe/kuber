package apis

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/sunweiwe/kuber/pkg/utils/kubertype"
	"github.com/sunweiwe/kuber/pkg/utils/pagination"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EventHandler struct {
	C client.Client
}

// @Tags        Agent.V1
// @Summary     获取Event列表数据
// @Description 获取Event列表数据
// @Accept      json
// @Produce     json
// @Param       order     query    string                                                             false "page"
// @Param       page      query    int                                                                false "page"
// @Param       size      query    int                                                                false "page"
// @Param       search    query    string                                                             false "search"
// @Param       namespace path     string                                                             true  "namespace"
// @Param       cluster   path     string                                                             true  "cluster"
// @Param       kind      query    string                                                             false "topkind"
// @Param       name      query    string                                                             false "topname"
// @Success     200       {object} handlers.ResponseStruct{Data=pagination.Pagination{List=[]object}} "Event"
// @Router      /v1/proxy/cluster/{cluster}/custom/core/v1/namespaces/{namespace}/events [get]
// @Security    JWT
func (h *EventHandler) List(c *gin.Context) {
	ns := c.Param("namespace")
	if ns == "_all" || ns == "_" {
		ns = ""
	}

	events := &v1.EventList{}
	if err := h.C.List(c.Request.Context(), events, client.InNamespace(ns)); err != nil {
		NotOK(c, err)
		return
	}

	objects := h.filterByKind(c, events.Items)

	pageData := pagination.NewTypedSearchSortPageResourceFromContext(c, objects)

	OK(c, pageData)
}

func (h *EventHandler) filterByKind(c *gin.Context, events []v1.Event) []v1.Event {
	kind := c.Query("kind")
	name := c.Query("name")
	if len(kind) == 0 || len(name) == 0 {
		return events
	}

	ns := c.Params.ByName("namespace")

	involvedObject := map[string]bool{
		involvedObjectKindName(kind, name): true,
	}

	switch kind {
	case kubertype.Deployment:
		dp := &appsv1.Deployment{}
		err := h.C.Get(c.Request.Context(), types.NamespacedName{Namespace: ns, Name: name}, dp)
		if err != nil {
			goto GOTO
		}
		replicaSets := &appsv1.ReplicaSetList{}
		err = h.C.List(c.Request.Context(), replicaSets, &client.ListOptions{
			LabelSelector: labels.SelectorFromSet(dp.Spec.Selector.MatchLabels),
		})
		if err != nil {
			goto GOTO
		}
		for _, rs := range replicaSets.Items {
			involvedObject[involvedObjectKindName(kubertype.ReplicaSet, rs.Name)] = true
		}
	case kubertype.DaemonSet:
		ds := &appsv1.DaemonSet{}
		err := h.C.Get(c.Request.Context(), types.NamespacedName{Namespace: ns, Name: name}, ds)
		if err != nil {
			goto GOTO
		}
		involvedObject[involvedObjectKindName(kubertype.DaemonSet, ds.Name)] = true
	case kubertype.StatefulSet:
		st := &appsv1.StatefulSet{}
		err := h.C.Get(c.Request.Context(), types.NamespacedName{Namespace: ns, Name: name}, st)
		if err != nil {
			goto GOTO
		}
		involvedObject[involvedObjectKindName(kubertype.StatefulSet, st.Name)] = true
	}

GOTO:
	var ret []v1.Event
	for _, evt := range events {
		if _, exist := involvedObject[involvedObjectKindName(evt.InvolvedObject.Kind, evt.InvolvedObject.Name)]; exist {
			ret = append(ret, evt)
		}
	}
	return ret
}

func involvedObjectKindName(kind, name string) string {
	return fmt.Sprintf("%s--%s", kind, name)
}
