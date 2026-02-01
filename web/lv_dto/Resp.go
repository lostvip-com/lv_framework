/*
 * Copyright 2025 lostvip
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

package lv_dto

type Resp struct {
	// 代码
	Code int `json:"code" example:"200"`
	// 数据集
	Data interface{} `json:"data"`
	// 消息
	Msg string `json:"msg"`
}
func (r *Resp) GetCode() int  {
	return r.Code
}
func (r *Resp) GetMsg() string  {
	return r.Msg
}
func (res *Resp) ReturnOK() *Resp {
	res.Code = 200
	res.Msg = "Success!"
	return res
}
func (res *Resp) RetData(data *interface{}) *Resp {
	res.Code = 200
	res.Data = data
	return res
}
func (res *Resp) ReturnError(code int) *Resp {
	res.Code = code
	return res
}

func (res *Resp) Fail(msg string) *Resp {
	res.Code = 1
	res.Msg = msg
	return res
}

func (res *Resp) Ok(data *interface{}) *Resp {
	res.Code = 200
	res.Data = data
	res.Msg = "success!"
	return res
}
