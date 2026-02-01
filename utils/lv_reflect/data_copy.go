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

package lv_reflect

import (
	"github.com/jinzhu/copier"
	"reflect"
)

func CopyProperties(fromValue interface{}, toValue interface{}) error {
	err := copier.CopyWithOption(toValue, fromValue, copier.Option{IgnoreEmpty: true})
	return err
}

func CopyProp(fromValue interface{}, toValue interface{}, ignoreEmpty bool) error {
	//dstType, dstValue := reflect.TypeOf(dst), reflect.ValueOf(dst)
	//srcType, srcValue := reflect.TypeOf(src), reflect.ValueOf(src)
	//// dst必须结构体指针类型
	//if dstType.Kind() != reflect.Ptr || dstType.Elem().Kind() != reflect.Struct {
	//	return errors.New("dst type should be a struct pointer")
	//}
	err := copier.CopyWithOption(toValue, fromValue, copier.Option{IgnoreEmpty: ignoreEmpty})
	return err
}

func IsMap(data interface{}) bool {
	t := reflect.TypeOf(data)
	return t.Kind() == reflect.Map
}

func Map2Struct(sourceMap map[any]any, destPtr any) error {
	err := copier.Copy(destPtr, sourceMap)
	return err
}
