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

package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/lostvip-com/lv_framework/lv_log"
	"github.com/lostvip-com/lv_framework/utils/lv_err"
	"github.com/lostvip-com/lv_framework/web/lv_dto"
	"github.com/spf13/cast"
	"net/http"
	"strings"
)

func RecoverError(c *gin.Context) {
	defer func() {
		err := recover()
		if err != nil {
			switch errTypeObj := err.(type) {
			case string:
				if strings.HasPrefix(errTypeObj, "{") {
					c.Header("Content-Type", "application/json; charset=utf-8")
					c.String(http.StatusOK, errTypeObj)
					c.Abort()
				} else {
					c.AbortWithStatusJSON(http.StatusOK, &lv_dto.CommonRes{Code: 500, Msg: errTypeObj})
				}
			case lv_dto.Resp: //封装过的
				c.AbortWithStatusJSON(http.StatusOK, errTypeObj)
			case error: // 原始的错误
				if gin.IsDebugging() {
					lv_err.PrintStackTrace(errTypeObj)
				}
				lv_log.Error(c, "CustomError XXXXXXXXXX: ", errTypeObj)
				c.AbortWithStatusJSON(http.StatusOK, &lv_dto.CommonRes{Code: 500, Msg: cast.ToString(errTypeObj)})
			default:
				lv_log.Error(c, "default CustomErrorXXXXXXXXXX: ", errTypeObj)
				c.AbortWithStatusJSON(http.StatusOK, &lv_dto.CommonRes{Code: 500, Msg: "未知错误!"})
			}
		} else {
			lv_log.Info(c, "-----------request over----------")
		}
	}()
	c.Next()
}
