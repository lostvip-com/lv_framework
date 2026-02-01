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

package lv_global

var IsDebug = false
var LogOutputType = "stdout,file"

const FLAG_DEL_YES = 1
const FLAG_DEL_NO = 0
const KEY_SWAGGER_OFF = "SwaggerOff"

var TraceId = "traceId"

// yaml key
const (
	KEY_CACHE_TYPE      = "application.cache-type"
	SESSION_TIMEOUT_KEY = "application.session.timeout" // 会话超时配置key
)
