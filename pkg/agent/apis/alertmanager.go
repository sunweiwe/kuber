package apis

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/alertmanager/client"
	"github.com/prometheus/client_golang/api"
	"k8s.io/client-go/kubernetes"
)

type AlertManagerHandler struct {
	client api.Client
	c      kubernetes.Interface
}

func NewAlertManagerClient(server string, k8sclient kubernetes.Interface) (*AlertManagerHandler, error) {
	client, err := api.NewClient(api.Config{Address: server})
	if err != nil {
		return nil, err
	}
	return &AlertManagerHandler{
		client: client,
		c:      k8sclient,
	}, nil
}

type alertQuery struct {
	Filter      string `form:"filter"`
	Receiver    string `form:"receiver"`
	Silenced    bool   `form:"silenced"`
	Inhibited   bool   `form:"inhibited"`
	Active      bool   `form:"active"`
	Unprocessed bool   `form:"unprocessed"`
}

// @Tags        Agent.V1
// @Summary     获取alertmanager中的告警数据
// @Description 获取alertmanager中的告警数据
// @Accept      json
// @Produce     json
// @Param       cluster     path     string                                               true  "cluster"
// @Param       filter      query    string                                               false "filter"
// @Param       receiver    query    string                                               false "receiver"
// @Param       silenced    query    bool                                                 false "silenced"
// @Param       inhibited   query    bool                                                 false "inhibited"
// @Param       active      query    bool                                                 false "active"
// @Param       unprocessed query    bool                                                 false "unprocessed"
// @Success     200         {object} handlers.ResponseStruct{Data=[]client.ExtendedAlert} "labelvalues"
// @Router      /v1/proxy/cluster/{cluster}/custom/alertmanager/v1/alerts [get]
// @Security    JWT
func (h *AlertManagerHandler) ListAlerts(c *gin.Context) {
	api := client.NewAlertAPI(h.client)
	query := &alertQuery{}
	_ = c.BindQuery(query)
	alerts, err := api.List(c.Request.Context(), query.Filter, query.Receiver, query.Silenced, query.Inhibited, query.Active, query.Unprocessed)
	if err != nil {
		NotOK(c, err)
	}
	OK(c, alerts)
}
