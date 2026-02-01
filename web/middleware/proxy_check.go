package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/lostvip-com/lv_framework/lv_conf"
	"github.com/lostvip-com/lv_framework/utils/lv_net"
)

func IfProxyForward() func(c *gin.Context) {
	return func(c *gin.Context) {
		// 客户端携带Token有三种方式 1.放在请求头 2.放在请求体 3.放在URI
		// 这里的具体实现方式要依据你的实际业务情况决定
		uri := c.Request.RequestURI
		isPorxyEnable := lv_conf.Config().IsProxyEnabled()
		if !isPorxyEnable { //不支持代理
			c.Next() // 后续的处理函数可以用过c.Get("username")来获取当前请求的用户信息
		}
		//支持代理
		mp := lv_conf.Config().GetProxyMap()
		pathSrc := lv_net.ParseUrlPath(uri)
		urlTarget := mp[pathSrc]
		if urlTarget != "" {
			lv_net.ProxyExact(c, urlTarget)
			return
		}

	}
}
