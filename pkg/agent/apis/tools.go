package apis

import "github.com/gin-gonic/gin"

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
