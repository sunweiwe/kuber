package handlers

import "github.com/gin-gonic/gin"

// TODO
func NotOK(c *gin.Context, err error) {

}

type ResponseStruct struct {
	Message   string
	Data      interface{}
	ErrorData interface{}
}
