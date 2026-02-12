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

package lv_conv

import (
	"github.com/spf13/cast"
	"strings"
)

func String(str any) string {
	return cast.ToString(str)
}
func Int64(str any) int64 {
	return cast.ToInt64(str)
}

func SubStr(str string, startIndex, endIndex int) string {
	rs := []rune(str)
	return string(rs[startIndex:endIndex])
}

func ToIntArray(str, split string) []int {
	result := make([]int, 0)
	if str == "" {
		return result
	}
	arr := strings.Split(str, split)
	if len(arr) > 0 {
		for i := range arr {
			if arr[i] != "" {
				result = append(result, cast.ToInt(arr[i]))
			}
		}
	}
	return result
}
func ToInt64Array(str, split string) []int64 {
	result := make([]int64, 0)
	if str == "" {
		return result
	}
	arr := strings.Split(str, split)
	if len(arr) > 0 {
		for i := range arr {
			if arr[i] != "" {
				result = append(result, cast.ToInt64(arr[i]))
			}
		}
	}
	return result
}
