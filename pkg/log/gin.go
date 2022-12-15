package log

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func DefaultGinLoggerMiddleware() gin.HandlerFunc {
	return NewGinLoggerMiddleware(GlobalLogger)
}

func NewGinLoggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	logger = logger.WithOptions(zap.AddCallerSkip(1))
	return func(ctx *gin.Context) {
		start := time.Now()
		ctx.Next()
		latency := time.Since(start)

		statusCode := ctx.Writer.Status()

		fields := []zap.Field{
			zap.String("method", ctx.Request.Method),
			zap.String("path", ctx.Request.URL.Path),
			zap.Int("code", statusCode),
			zap.Duration("latency", latency),
		}

		if len(ctx.Errors) != 0 {
			logger.Error(ctx.Errors.String(), fields...)
			return
		}

		if statusCode >= http.StatusInternalServerError {
			logger.Error(http.StatusText(statusCode), fields...)
			return
		}

		if statusCode >= http.StatusBadRequest {
			logger.Warn(http.StatusText(statusCode), fields...)
			return
		}
		logger.Debug(http.StatusText(statusCode), fields...)
	}
}
