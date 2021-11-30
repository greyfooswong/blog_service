package middleware

import (
	"blog-service/pkg/app"
	"blog-service/pkg/errcode"
	"blog-service/pkg/limiter"
	"github.com/gin-gonic/gin"
)

func RateLimiter(l limiter.LimiterIface) gin.HandlerFunc {
	return func(context *gin.Context) {
		key := l.Key(context)
		if bucket, ok := l.GetBucket(key); ok {
			count := bucket.TakeAvailable(1)
			if count == 0 {
				response := app.NewResponse(context)
				response.ToErrorResponse(errcode.TooManyRequests)
				context.Abort()
				return
			}
		}
		context.Next()
	}
}
