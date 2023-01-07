package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/sunweiwe/kuber/pkg/i18n"
	"github.com/sunweiwe/kuber/pkg/service/models"
	"github.com/sunweiwe/kuber/pkg/service/models/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	MessageOK           = "ok"
	MessageNotFound     = "not found"
	MessageError        = "err"
	MessageForbidden    = "forbidden"
	MessageUnauthorized = "unauthorized"
)

func Response(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(code, ResponseStruct{Message: message, Data: data})
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, ResponseStruct{Message: MessageOK, Data: data})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, ResponseStruct{Message: MessageOK, Data: data})
}
func NoContent(c *gin.Context, data interface{}) {
	c.JSON(http.StatusNoContent, ResponseStruct{Message: MessageOK, Data: data})
}

func Forbidden(c *gin.Context, data interface{}) {
	c.JSON(http.StatusForbidden, ResponseStruct{Message: MessageForbidden, Data: data})
}

func Unauthorized(c *gin.Context, data interface{}) {
	c.JSON(http.StatusUnauthorized, ResponseStruct{Message: MessageUnauthorized, Data: data})
}

func errResponse(errData interface{}) ResponseStruct {
	return ResponseStruct{Message: MessageError, ErrorData: errData}
}

func NotOK(c *gin.Context, err error) {
	defer func() {
		c.Errors = append(c.Errors, &gin.Error{
			Err:  err,
			Type: gin.ErrorTypeAny,
		})
	}()
	if errs, ok := err.(validator.ValidationErrors); ok {
		vErrors := []string{}
		for _, e := range errs {
			vErrors = append(vErrors, e.Translate(validate.Get().Translator))
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, errResponse(strings.Join(vErrors, ";")))
		return
	}

	if err, ok := err.(*errors.StatusError); ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, errResponse(err.Error()))
		return
	}

	if rpcErr, ok := status.FromError(err); ok {
		if rpcErr.Code() == codes.NotFound {
			c.AbortWithStatusJSON(http.StatusNotFound, errResponse(err.Error()))
			return
		}
	}

	if models.NotFound(err) {
		message := i18n.Sprintf(context.TODO(), "The object or the parent object is not found")
		c.AbortWithStatusJSON(http.StatusNotFound, errResponse(message))
		return
	}
	message := models.GetErrMessage(err)
	c.AbortWithStatusJSON(http.StatusBadRequest, errResponse(message))
}

type ResponseStruct struct {
	Message   string
	Data      interface{}
	ErrorData interface{}
}
