package apis

import (
	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	allNamespace = "_all"
)

func paramFromHeaderOrQuery(c *gin.Context, key, defaultV string) string {
	hv := c.Request.Header.Get(key)
	if hv != "" {
		return hv
	}

	qv := c.Query(key)
	if qv != "" {
		return qv
	}

	return defaultV
}

func fieldSelector(c *gin.Context) (fields.Selector, bool) {
	fieldSelectorStr := c.Query("fieldSelector")
	if len(fieldSelectorStr) == 0 {
		return nil, false
	}
	sel, err := fields.ParseSelector(fieldSelectorStr)
	if err != nil {
		return nil, false
	}
	return sel, true
}

func labelSelector(c *gin.Context) labels.Selector {
	labelsMap := c.QueryMap("labels")
	sel := labels.SelectorFromSet(labelsMap)
	return sel
}
