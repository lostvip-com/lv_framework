package lv_net

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/lostvip-com/lv_framework/lv_log"
)

// ProxyExact  精确代理：自动最长前缀匹配 + 路径重写 + 零数据丢失
// 调用示例：ProxyExact(c, "https://order-svc:8443/api/v1")
func ProxyExact(c *gin.Context, targetFull string) {
	lv_log.Debug("===== ProxyExact start =====")
	defer lv_log.Debug("===== ProxyExact over  =====")

	target, err := url.Parse(targetFull)
	if err != nil {
		lv_log.Errorf("invalid target url: %v", err)
		c.String(http.StatusBadGateway, "invalid target")
		c.Abort()
		return
	}

	// 1. 克隆请求（连带 body）
	req := c.Request.Clone(c.Request.Context())

	// 2. Director：只改必要字段，其余全保留
	director := func(out *http.Request) {
		out.URL.Scheme = target.Scheme
		out.URL.Host = target.Host
		out.URL.Path = target.Path          // 已包含重写后的路径
		out.URL.RawQuery = req.URL.RawQuery // 保留原始 query
		out.Host = target.Host              // Host 头
		// 可选：透传真实客户端 IP
		out.Header.Set("X-Forwarded-For", c.ClientIP())
	}

	proxy := &httputil.ReverseProxy{Director: director}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		lv_log.Errorf("proxy error: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}
	proxy.ServeHTTP(c.Writer, req)
	c.Abort()
}

func ParseUrlPath(uri string) string {
	u, _ := url.Parse(uri) // uri 可以是 "/api/order?a=b" 或 "http://baidu.com/api/order?a=b"
	path := u.Path         // 得到干净的 "/api/order"
	return path
}
