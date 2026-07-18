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

package lv_err

import (
	"database/sql"
	"errors"
	"fmt"
	"runtime"

	"github.com/lostvip-com/lv_framework/lv_log"
	"github.com/lostvip-com/lv_framework/utils/lv_json"
	"github.com/lostvip-com/lv_framework/web/lv_dto"
	"gorm.io/gorm"
)

// IfErrPanic 当 error 不为 nil 时 panic。
func IfErrPanic(err error) {
	if err != nil {
		panic(err)
	}
}

// Assert1 条件为真时 panic 一个 lv_dto.Resp（业务断言响应专用）。
func Assert1(conditionYes bool, msg string) {
	if conditionYes {
		var res lv_dto.Resp
		res.Msg = msg
		res.Code = 1
		json := lv_json.ToJsonStr(res)
		panic(json)
	}
}

// HasErrorMsg 当 error 不为 nil 时 panic 指定 msg 的 lv_dto.Resp。
// Deprecated: 使用 WrapErr 返回 error。
func HasErrorMsg(err error, msg string) {
	if err != nil {
		if msg == "" {
			msg = err.Error()
		}
		_, file, line, _ := runtime.Caller(1)
		lv_log.Error(fmt.Sprintf("%s:%v error: %#v", file, line, err))
		var res lv_dto.Resp
		res.Msg = msg
		res.Code = 1
		panic(res)
	}
}

// HasError1 当 error 不为 nil 时 panic 含 error 信息的 lv_dto.Resp。
// Deprecated: 使用 CheckErr。
func HasError1(err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		lv_log.Error(fmt.Sprintf("%s:%v error: %#v", file, line, err))
		var res lv_dto.Resp
		res.Msg = err.Error()
		res.Code = 1
		panic(res)
	}
}

func IsRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, sql.ErrNoRows)
}
