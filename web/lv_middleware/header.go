package lv_middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lostvip-com/lv_framework/lv_global"
)

// ...

func SetTraceId(c *gin.Context) {

	traceId := c.GetHeader(lv_global.TraceId)
	if traceId == "" {
		traceId := strings.ReplaceAll(uuid.New().String(), "-", "")
		c.Header(lv_global.TraceId, traceId)
		c.Request.Header.Set(lv_global.TraceId, traceId)
		c.Set(lv_global.TraceId, traceId)
		//c.Writer.Header().Set(key, traceId)
	}
	c.Next()
}

// NoCache is a lv_middleware function that appends headers
// to prevent the client from caching the HTTP response.
func NoCache(c *gin.Context) {
	c.Header("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate, value")
	c.Header("Expires", "Thu, 01 Jan 1970 00:00:00 GMT")
	c.Header("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
	c.Next()
}

// Options 全局跨域中间件：对每个响应都附加跨域响应头；
// 预检(OPTIONS)请求直接返回，不进入后续业务处理链。
func Options(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")

	// 按客户端实际请求的头部回显，自定义头也能通过预检；未携带时退回默认清单
	allowHeaders := c.Request.Header.Get("Access-Control-Request-Headers")
	if allowHeaders == "" {
		allowHeaders = "authorization, origin, content-type, accept, x-requested-with"
	}
	c.Header("Access-Control-Allow-Headers", allowHeaders)

	c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Type")
	c.Header("Access-Control-Max-Age", "86400")

	if c.Request.Method == "OPTIONS" {
		c.Header("Allow", "HEAD,GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Content-Type", "application/json")
		c.AbortWithStatus(200)
		return
	}
	c.Next()
}

// Secure is a lv_middleware function that appends security
// and resource access headers.
func Secure(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	//c.Header("X-Frame-Options", "DENY")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("X-XSS-Protection", "1; mode=block")
	if c.Request.TLS != nil {
		c.Header("Strict-Transport-Security", "max-age=31536000")
	}

	// Also consider adding Content-Security-Policy headers
	// c.Header("Content-Security-Policy", "script-src 'self' https://cdnjs.cloudflare.com")
}
