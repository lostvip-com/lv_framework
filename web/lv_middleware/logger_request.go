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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lostvip-com/lv_framework/lv_log"
)

// LoggerURI 日志记录URI
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

		// 请求路由
		reqUri := c.Request.RequestURI

		// 状态码
		statusCode := c.Writer.Status()

		// 请求IP
		clientIP := c.ClientIP()

		// 日志格式 - 避免创建 map，直接格式化输出
		lv_log.Infof("status=%d latency=%s ip=%s method=%s uri=%s",
			statusCode,
			latencyTime,
			clientIP,
			reqMethod,
			reqUri,
		)

		// 禁用日志写入数据库的功能 ssz20210702
		//if c.Request.Method != "GET" && c.Request.Method != "OPTIONS" && lv_conf.LoggerConfig.EnabledDB {
		//	SetDBOperLog(c, clientIP, statusCode, reqUri, reqMethod, latencyTime)
		//}
	}
}
