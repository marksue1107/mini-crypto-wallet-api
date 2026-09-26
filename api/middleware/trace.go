package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const TraceIDHeader = "X-Trace-ID"
const TraceIDKey = "trace_id"

// TraceMiddleware 添加追蹤 ID 中間件
func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(TraceIDHeader)
		if traceID == "" {
			traceID = uuid.New().String()
		}

		c.Set(TraceIDKey, traceID)
		c.Header(TraceIDHeader, traceID)
		c.Next()
	}
}
