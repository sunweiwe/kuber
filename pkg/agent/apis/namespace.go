package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/sunweiwe/kuber/pkg/api/kuber"
	"github.com/sunweiwe/kuber/pkg/utils/pagination"
	"github.com/sunweiwe/kuber/pkg/utils/slice"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type NamespaceHandler struct {
	C client.Client
}

var forbiddenBindNamespaces = []string{
	"kube-system",
	"istio-system",
	kuber.NamespaceSystem,
	kuber.NamespaceLocal,
	kuber.NamespaceInstaller,
	kuber.NamespaceMonitor,
	kuber.NamespaceLogging,
	kuber.NamespaceGateway,
}

// @Tags        Agent.V1
// @Summary     获取可以绑定的环境的namespace列表数据
// @Description 获取可以绑定的环境的namespace列表数据
// @Accept      json
// @Produce     json
// @Param       order   query    string                                                             false "page"
// @Param       search  query    string                                                             false "search"
// @Param       page    query    int                                                                false "page"
// @Param       size    query    int                                                                false "page"
// @Param       cluster path     string                                                             true  "cluster"
// @Success     200     {object} handlers.ResponseStruct{Data=pagination.Pagination{List=[]object}} "Namespace"
// @Router      /v1/proxy/cluster/{cluster}/custom/core/v1/namespaces [get]
// @Security    JWT

func (h *NamespaceHandler) List(c *gin.Context) {
	nsList := &corev1.NamespaceList{}
	selector := labels.NewSelector()
	req, _ := labels.NewRequirement(kuber.LabelEnvironment, selection.DoesNotExist, []string{})
	listOptions := &client.ListOptions{
		LabelSelector: selector.Add(*req),
	}
	if err := h.C.List(c.Request.Context(), nsList, listOptions); err != nil {
		NotOK(c, err)
		return
	}
	objects := []corev1.Namespace{}
	for _, obj := range nsList.Items {
		if slice.ContainStr(forbiddenBindNamespaces, obj.Name) {
			objects = append(objects, obj)
		}
	}

	pageData := pagination.NewTypedSearchSortPageResourceFromContext(c, objects)
	OK(c, pageData)
}
