package apis

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/sunweiwe/kuber/pkg/service/handlers"
)

type prometheusHandler struct {
	client api.Client
}

func NewPrometheusHandler(server string) (*prometheusHandler, error) {
	client, err := api.NewClient(api.Config{Address: server})
	if err != nil {
		return nil, err
	}
	return &prometheusHandler{client: client}, nil
}

// @Tags        Agent.V1
// @Summary     Prometheus Vector
// @Description Prometheus Vector
// @Accept      json
// @Produce     json
// @Param       cluster path     string                               true "cluster"
// @Param       query   query    string                               false "query"
// @Param       nullable query    bool                                false "nullable"
// @Success     200     {object} handlers.ResponseStruct{Data=object} "vector"
// @Router      /v1/proxy/cluster/{cluster}/custom/prometheus/v1/vector [get]
// @Security    JWT
func (p *prometheusHandler) Vector(c *gin.Context) {
	query := c.Query("query")

	v1api := v1.NewAPI(p.client)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	obj, _, err := v1api.Query(ctx, query, time.Now())
	if err != nil {
		NotOK(c, err)
		return
	}
	if nullable, _ := strconv.ParseBool(c.Query("nullable")); nullable {
		if obj.String() == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, handlers.ResponseStruct{
				Message:   "空查询",
				Data:      nil,
				ErrorData: "空查询",
			})
			return
		}
	}
	OK(c, obj)
}
