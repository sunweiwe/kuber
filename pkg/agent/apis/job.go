package apis

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sunweiwe/kuber/pkg/agent/cluster"
	"github.com/sunweiwe/kuber/pkg/utils/pagination"
	v1 "k8s.io/api/batch/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type JobHandler struct {
	C       client.Client
	cluster cluster.Interface
}

// @Tags        Agent.V1
// @Summary     获取Job列表数据
// @Description 获取Job列表数据
// @Accept      json
// @Produce     json
// @Param       order     query    string                                                           false "page"
// @Param       search    query    string                                                           false "search"
// @Param       page      query    int                                                              false "page"
// @Param       size      query    int                                                              false "page"
// @Param       namespace path     string                                                           true  "namespace"
// @Param       cluster   path     string                                                           true  "cluster"
// @Param       kind      query    string                                                           false "kind(cronjob)"
// @Param       name      query    string                                                           false "name"
// @Success     200       {object} handlers.ResponseStruct{Data=pagination.PageData{List=[]object}} "Job"
// @Router      /v1/proxy/cluster/{cluster}/custom/batch/v1/namespaces/{namespace}/jobs [get]
// @Security    JWT
func (h *JobHandler) List(c *gin.Context) {
	ns := c.Param("namespace")
	jobList := &v1.JobList{}
	if ns == "_all" || ns == "_" {
		ns = ""
	}

	listOptions := &client.ListOptions{
		Namespace:     ns,
		LabelSelector: labelSelector(c),
	}
	if err := h.C.List(c.Request.Context(), jobList, listOptions); err != nil {
		NotOK(c, err)
		return
	}

	objects := h.filterJobByName(c, jobList.Items)
	page := pagination.NewTypedSearchSortPageResourceFromContext(c, objects)

	if watch, _ := strconv.ParseBool(c.Query("watch")); watch {
		c.SSEvent("data", page)
		c.Writer.Flush()

		WatchEvents(c, h.cluster, jobList, listOptions)
		return
	} else {
		OK(c, page)
	}
}

func (h *JobHandler) filterJobByName(c *gin.Context, jobs []v1.Job) []v1.Job {
	kind := c.Query("kind")
	name := c.Query("name")

	if len(kind) == 0 || len(name) == 0 {
		return jobs
	}
	var ret []v1.Job
	for _, job := range jobs {
		for _, owner := range job.OwnerReferences {
			if strings.EqualFold(owner.Kind, kind) && owner.Name == name {
				ret = append(ret, job)
			}
		}
	}
	return ret
}
