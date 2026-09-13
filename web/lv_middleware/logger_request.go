/*
 * Copyright 2019 lostvip
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package lv_middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lostvip-com/lv_framework/lv_log"
)

// LoggerURI 记录请求访问日志
func LoggerURI() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		startTime := time.Now()

		// 处理请求
		c.Next()

		// 结束时间
		endTime := time.Now()

		// 执行时间
		latencyTime := endTime.Sub(startTime)

		// 请求方式
		reqMethod := c.Request.Method

		// 请求路由（脱敏，避免 token/password 等明文落日志）
		reqUri := sanitizeURI(c.Request.RequestURI)

		// 状态码
		statusCode := c.Writer.Status()

		// 请求IP
		clientIP := c.ClientIP()

		// traceId
		traceId := getTraceId(c)

		lv_log.Infof("status=%d latency=%s ip=%s method=%s uri=%s traceId=%s",
			statusCode,
			latencyTime,
			clientIP,
			reqMethod,
			reqUri,
			traceId,
		)

		// 禁用日志写入数据库的功能 ssz20210702
		//if c.Request.Method != "GET" && c.Request.Method != "OPTIONS" && lv_conf.LoggerConfig.EnabledDB {
		//	SetDBOperLog(c, clientIP, statusCode, reqUri, reqMethod, latencyTime)
		//}
	}
}

// sanitizeURI 对请求 URI query 中的敏感参数（token/password/secret 等）脱敏
func sanitizeURI(uri string) string {
	idx := strings.Index(uri, "?")
	if idx < 0 {
		return uri
	}
	path := uri[:idx+1]
	query := uri[idx+1:]
	if query == "" {
		return uri
	}
	sensitive := map[string]bool{
		"token": true, "password": true, "pwd": true, "secret": true,
		"access_token": true, "refresh_token": true, "authorization": true,
	}
	pairs := strings.Split(query, "&")
	for i, p := range pairs {
		kv := strings.SplitN(p, "=", 2)
		if len(kv) == 2 && sensitive[strings.ToLower(kv[0])] {
			pairs[i] = kv[0] + "=***"
		}
	}
	return path + strings.Join(pairs, "&")
}
