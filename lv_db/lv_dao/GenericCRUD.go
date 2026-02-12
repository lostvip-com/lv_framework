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

package lv_dao

import (
	"gorm.io/gorm"
)

// CRUD 接口定义了通用的CRUD操作
type CRUD[T any] interface {
	Create(db *gorm.DB, model *T) error
	Find(db *gorm.DB, out **T, id uint) error
	Update(db *gorm.DB, model *T) error
	Delete(db *gorm.DB, model *T) error
}

// GenericCRUD 是CRUD接口的一个泛型实现
type GenericCRUD[T any] struct {
	db *gorm.DB
}

// NewGenericCRUD 创建一个新的GenericCRUD实例
func NewGenericCRUD[T any](db *gorm.DB) *GenericCRUD[T] {
	return &GenericCRUD[T]{db: db}
}

// Create 创建一条记录
func (g *GenericCRUD[T]) Create(model *T) error {
	return g.db.Create(model).Error
}

// Save 创建一条记录
func (g *GenericCRUD[T]) Save(model *T) error {
	return g.db.Save(model).Error
}

// FindById 根据ID查找记录
func (g *GenericCRUD[T]) FindById(out *T, id uint) error {
	return g.db.First(out, id).Error
}

// FindList 根据ID查找记录
func (g *GenericCRUD[T]) FindList(list []T, start int, pageSize int, condition string, args ...any) error {
	result := g.db.Where(condition, args...).Offset(start).Limit(pageSize).Find(list)
	return result.Error
}

// FindFirst 根据ID查找记录
func (g *GenericCRUD[T]) FindFirst(out *T, condition string, args ...any) error {
	result := g.db.Where(condition, args...).First(out)
	return result.Error
}

// Update 更新记录
func (g *GenericCRUD[T]) Update(model *T) error {
	return g.db.Save(model).Error
}

// Delete 删除记录
func (g *GenericCRUD[T]) Delete(model *T) error {
	return g.db.Delete(model).Error
}
