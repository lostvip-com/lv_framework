// Copyright 2019 gf Author(https://github.com/gogf/gf). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

package lv_arr

import (
	"reflect"

	"github.com/lostvip-com/lv_framework/lv_log"
)

func GetArrayOfKV[K, V comparable](rowMap map[K]V) ([]K, []V) {
	listK := make([]K, len(rowMap))
	listV := make([]V, len(rowMap))
	for k, v := range rowMap {
		listK = append(listK, k)
		listV = append(listV, v)
	}
	return listK, listV
}
func GetArrayByKeys[K, V comparable](rowMap map[K]V, keys []K) []V {
	listVal := make([]V, 0)
	for _, keyVal := range keys {
		v := rowMap[keyVal]
		lv_log.Debug("--======>", keyVal, ":", v)
		listVal = append(listVal, v)
	}
	return listVal
}

func Contains[T comparable](slice []T, item T) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// IsArray checks whether given value is array/slice.
// Note that it uses reflect internally implementing this feature.
// It's designed to handle any input type and determine its nature at runtime.
func IsCollection(value interface{}) bool {
	rv := reflect.ValueOf(value)
	kind := rv.Kind()
	if kind == reflect.Ptr {
		rv = rv.Elem()
		kind = rv.Kind()
	}
	switch kind {
	case reflect.Array, reflect.Slice:
		return true
	default:
		return false
	}
}

// Remove 从切片中删除所有等于 element 的元素（泛型版本）
// 支持任何可比较类型的切片，使用原地删除提高内存效率
func Remove[T comparable](slice []T, element T) []T {
	idx := 0 // 慢指针，指向当前有效元素的下一个位置
	for _, v := range slice {
		if v != element {
			slice[idx] = v // 将不等于目标元素的值移动到前面
			idx++
		}
	}
	return slice[:idx] // 返回包含有效元素的切片部分
}
