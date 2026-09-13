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
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lostvip-com/lv_framework/lv_global"
	"github.com/lostvip-com/lv_framework/lv_log"
	"github.com/lostvip-com/lv_framework/utils/lv_err"
	"github.com/lostvip-com/lv_framework/web/lv_dto"
)

// RecoverError 捕获 panic 并转换为 HTTP 响应。
// 安全原则：对外只返回中性提示（避免泄漏 SQL/内部路径），原始错误带 traceId 落日志。
func RecoverError(c *gin.Context) {
	defer func() {
		err := recover()
		if err == nil {
			return
		}
		traceId := getTraceId(c)
		switch errTypeObj := err.(type) {
		case lv_dto.Resp: // 业务受控返回
			c.AbortWithStatusJSON(http.StatusOK, errTypeObj)
		case string:
			// 业务主动抛出的提示文案
			lv_log.Error(fmt.Sprintf("[traceId=%s] panic(string): %v", traceId, errTypeObj))
			c.AbortWithStatusJSON(http.StatusOK, &lv_dto.CommonRes{Code: 500, Msg: errTypeObj})
		case error:
			// 原始错误：对外中性提示，详情带 traceId 落日志
			if gin.IsDebugging() {
				lv_err.PrintStackTrace(errTypeObj)
			}
			lv_log.Error(fmt.Sprintf("[traceId=%s] panic(error): %v", traceId, errTypeObj))
			c.AbortWithStatusJSON(http.StatusOK, &lv_dto.CommonRes{Code: 500, Msg: "系统繁忙，请稍后重试"})
		default:
			lv_log.Error(fmt.Sprintf("[traceId=%s] panic(unknown): %v", traceId, errTypeObj))
			c.AbortWithStatusJSON(http.StatusOK, &lv_dto.CommonRes{Code: 500, Msg: "系统繁忙，请稍后重试"})
		}
	}()
	c.Next()
}

// getTraceId 从 gin.Context 取 traceId（由 SetTraceId 中间件写入）
func getTraceId(c *gin.Context) string {
	if v, ok := c.Get(lv_global.TraceId); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
