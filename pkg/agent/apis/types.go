package apis

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sunweiwe/kuber/pkg/log"
	"github.com/sunweiwe/kuber/pkg/service/handlers"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
)

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, handlers.ResponseStruct{Message: "ok", Data: data})
}

func NotOK(c *gin.Context, err error) {
	log.Errorf("Not OK: %v", err)
	statusCode := http.StatusBadRequest

	statusErr := &apiErrors.StatusError{}
	if errors.As(err, &statusErr) {
		c.AbortWithStatusJSON(int(statusErr.Status().Code), handlers.ResponseStruct{Message: err.Error(), ErrorData: statusErr})
		return
	}

	c.AbortWithStatusJSON(statusCode, handlers.ResponseStruct{Message: err.Error(), ErrorData: err})
}
