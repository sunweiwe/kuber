package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/sunweiwe/kuber/pkg/log"
	"github.com/sunweiwe/kuber/pkg/service/handlers"
	"github.com/sunweiwe/kuber/pkg/utils/httpsigs"
)

func SignerMiddleware() func(c *gin.Context) {
	signer := httpsigs.GetSigner()
	signer.AddWhiteList("/alert")
	signer.AddWhiteList("/alert")
	signer.AddWhiteList("/healthz")

	return func(c *gin.Context) {
		if err := signer.Validate(c.Request); err != nil {
			log.Error(err, "signer")
			handlers.Forbidden(c, err)
			c.Abort()
		}
	}
}
