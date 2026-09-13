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

// //////////////////////////////////////////////////////////////////////////////////////
// from Dotsql
// It is not an ORM, it is not a query builder.
// Dotsql is a library that helps you keep sql files in one place and use it with ease.
// For more usage examples see https://github.com/qustavo/dotsql
// /////////////////////////////////////////////////////////////////////////////////////
package lv_batis

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/lostvip-com/lv_framework/utils/lv_file"
	"github.com/lostvip-com/lv_framework/utils/lv_reflect"
	"github.com/lostvip-com/lv_framework/utils/lv_tpl"
	"github.com/morrisxyang/xreflect"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

// Execer is an interface used by Exec.
type Execer interface {
	Exec(query string, args ...interface{}) *gorm.DB
}

// ExecerContext is an interface used by ExecContext.
type ExecerContext interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

type LvBatis struct {
	Queries map[string]string
	Vars    map[string]map[string]any // 只读：各 tag 的变量名集合（不再作为可写参数容器）
	TplFile string
}

// sqlFileCache 缓存已解析的 LvBatis（key=绝对路径）。Queries/Vars 加载后只读，可安全并发共享
var sqlFileCache sync.Map

/**
 * 从mapper目录解析sql文件，命中缓存直接返回（不再每次磁盘IO+正则解析）
 */
func NewInstance(relativePath string) (*LvBatis, error) {
	basePath, _ := os.Getwd()
	absolutePath := filepath.Join(basePath, "resources", "mapper", relativePath)
	if v, ok := sqlFileCache.Load(absolutePath); ok {
		return v.(*LvBatis), nil
	}
	dot, err := LoadFromFile(absolutePath)
	if err != nil {
		return nil, err
	}
	dot.TplFile = relativePath
	// LoadOrStore：并发首调时只有一个 dot 落盘，其余拿到同一个
	actual, _ := sqlFileCache.LoadOrStore(absolutePath, dot)
	return actual.(*LvBatis), nil
}

func (d *LvBatis) GetSql(tagName string, params interface{}) (string, error) {
	query, err := d.LookupQuery(tagName)
	if err != nil {
		return "", err
	}
	if query == "" {
		return "", fmt.Errorf("%s: tag [%s] 内容为空", d.TplFile, tagName)
	}
	//动态解析
	sql, err := lv_tpl.ParseTemplateStr(query, params)
	if err != nil {
		return "", fmt.Errorf("%s 解析 tag [%s] 失败: %w（可能：1.使用了参数对象中不存在的属性 2.template语法错误）", d.TplFile, tagName, err)
	}
	if sql == "" {
		return "", fmt.Errorf("%s: tag [%s] 渲染后 sql 为空", d.TplFile, tagName)
	}
	return sql, nil
}

// resolve 解析分页参数并渲染基础SQL（局部变量传递，不再依赖实例共享状态 CurrBaseSql/Vars）
func (d *LvBatis) resolve(tagName string, params interface{}) (baseSql string, pageNum, pageSize int, sqlParams map[string]any, err error) {
	var pNum, pSize any
	sqlParams = make(map[string]any) // 每次新建，避免写共享 Vars 导致并发 map writes / 脏数据残留
	paramType := reflect.TypeOf(params).Kind()
	if paramType == reflect.Map {
		paramMap := params.(map[string]interface{})
		pNum = paramMap["pageNum"]
		pSize = paramMap["pageSize"]
		for key, value := range paramMap {
			sqlParams[key] = value
		}
	} else {
		pNum, _ = xreflect.FieldValue(params, "PageNum")
		pSize, _ = xreflect.FieldValue(params, "PageSize")
		lv_reflect.CopyProperties2Map(params, sqlParams)
	}
	pageSize = normalizePageSize(cast.ToInt(pSize))
	pageNum = normalizePageNum(cast.ToInt(pNum))
	baseSql, err = d.GetSql(tagName, sqlParams)
	return baseSql, pageNum, pageSize, sqlParams, err
}

// normalizePageSize 限制分页大小：<=0 或超过上限时回退到上限，防止恶意大页打爆DB
func normalizePageSize(size int) int {
	const maxPageSize = 1000
	if size <= 0 || size > maxPageSize {
		return maxPageSize
	}
	return size
}

// normalizePageNum 页码兜底为 1，避免负数/0 导致非法 offset
func normalizePageNum(num int) int {
	if num <= 0 {
		return 1
	}
	return num
}

func (d *LvBatis) GetLimitSqlParams(tagName string, params interface{}) (string, map[string]any, error) {
	baseSql, pageNum, pageSize, sqlParams, err := d.resolve(tagName, params)
	if err != nil {
		return "", nil, err
	}
	start := int64(pageSize) * (int64(pageNum) - 1)
	sql := baseSql + " limit " + cast.ToString(start) + "," + cast.ToString(pageSize)
	return sql, sqlParams, nil
}

func (d *LvBatis) GetLimitSql(tagName string, params interface{}) (string, error) {
	baseSql, pageNum, pageSize, _, err := d.resolve(tagName, params)
	if err != nil {
		return "", err
	}
	start := int64(pageSize) * (int64(pageNum) - 1)
	// 兼容 mysql/postgresql 的分页方式
	return baseSql + " limit " + cast.ToString(pageSize) + " offset " + cast.ToString(start), nil
}

func (d *LvBatis) GetCountSql(tagName string, params interface{}) (string, error) {
	baseSql, _, _, _, err := d.resolve(tagName, params)
	if err != nil {
		return "", err
	}
	return " select count(*) from (" + removeOrderByClause(baseSql) + ") t ", nil
}

// removeOrderByClause 粗略去除 order by 子句（大小写不敏感）。
// 注意：SQL 字面量内含 "order by" 仍可能误截，彻底方案需 SQL 词法分析。
func removeOrderByClause(sql string) string {
	lower := strings.ToLower(sql)
	idx := strings.Index(lower, " order by ")
	if idx > 0 {
		return sql[:idx]
	}
	return sql
}

func (d *LvBatis) GetPageSql(tagName string, params any) (string, string, error) {
	baseSql, pageNum, pageSize, _, err := d.resolve(tagName, params)
	if err != nil {
		return "", "", err
	}
	start := int64(pageSize) * (int64(pageNum) - 1)
	limitSql := baseSql + " limit " + cast.ToString(pageSize) + " offset " + cast.ToString(start)
	countSql := " select count(*) from (" + removeOrderByClause(baseSql) + ") t "
	return limitSql, countSql, nil
}

func (d *LvBatis) LookupQuery(name string) (query string, err error) {
	query, ok := d.Queries[name]
	if !ok {
		err = fmt.Errorf("sql: '%s' could not be found", name)
	}

	return
}

// Exec 默认只能执行单条sql，除非mysql配置为allowMultiQueries=true
func (d *LvBatis) Exec(db Execer, name string, args ...interface{}) (*gorm.DB, error) {
	query, err := d.LookupQuery(name)
	if err != nil {
		return nil, err
	}
	return db.Exec(query, args...), err
}

// ExecMultiSqlInTransaction 在事务中执行多个SQL语句
func (d *LvBatis) ExecMultiSqlInTransaction(db *gorm.DB, name string, args ...interface{}) (*gorm.DB, error) {
	query, err := d.LookupQuery(name)
	if err != nil {
		return nil, err
	}
	// 开启事务
	tx := db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var result *gorm.DB
	// 按分号拆分SQL语句
	statements := strings.Split(query, ";")
	for _, stmt := range statements {
		trimmedStmt := strings.TrimSpace(stmt)
		if trimmedStmt != "" {
			result = tx.Exec(trimmedStmt, args...)
			if result.Error != nil {
				tx.Rollback()
				return nil, result.Error
			}
		}
	}
	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return result, nil
}

// ExecContext is a wrapper for database/sql's ExecContext(), using dotsql named query.
func (d *LvBatis) ExecContext(ctx context.Context, db ExecerContext, name string, args ...interface{}) (sql.Result, error) {
	query, err := d.LookupQuery(name)
	if err != nil {
		return nil, err
	}

	return db.ExecContext(ctx, query, args...)
}

// GetRawSql returns the query, everything after the --name tag
func (d *LvBatis) GetRawSql(name string) (string, error) {
	return d.LookupQuery(name)
}

// GetQueryMap returns a map[string]string of loaded Queries
func (d *LvBatis) GetQueryMap() map[string]string {
	return d.Queries
}

func (d *LvBatis) getTplFile() string {
	return d.TplFile
}

// Load imports sql Queries from any io.Reader.
func Load(r io.Reader) (*LvBatis, error) {
	scanner := &Scanner{}
	queries := scanner.Run(bufio.NewScanner(r))
	varMap := parseVarName(queries)
	dotSql := &LvBatis{
		Queries: queries,
		Vars:    varMap,
	}

	return dotSql, nil
}

func parseVarName(funSql map[string]string) map[string]map[string]any {
	//re := regexp.MustCompile(`\.\w+`)
	// 使用正则匹配模板中的变量
	re := regexp.MustCompile(`{{[^{}]*\.\s*([a-zA-Z_][a-zA-Z0-9_]*)[^{}]*}}`)
	mp := make(map[string]map[string]any)
	for funcKey, sql := range funSql {
		matches := re.FindAllStringSubmatch(sql, -1)
		varMap := make(map[string]any) //变量去重
		mp[funcKey] = varMap
		for _, match := range matches {
			if len(match) > 1 {
				key := match[1]
				varMap[key] = nil
			}
		} //end for

	} //end for
	return mp
}

// LoadFromFile imports SQL Queries from the file.
func LoadFromFile(sqlFile string) (*LvBatis, error) {
	if !lv_file.IsFileExist(sqlFile) {
		return nil, errors.New("file not be found! " + sqlFile)
	}
	f, err := os.Open(sqlFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return Load(f)
}

// LoadFromString imports SQL Queries from the string.
func LoadFromString(sql string) (*LvBatis, error) {
	buf := bytes.NewBufferString(sql)
	return Load(buf)
}

// Merge takes one or mord *LvBatis and merge its Queries
// It's in-order, so the last source will override Queries with the same name
// in the previous arguments if any.
func Merge(dots ...*LvBatis) *LvBatis {
	queries := make(map[string]string)

	for _, dot := range dots {
		for k, v := range dot.GetQueryMap() {
			queries[k] = v
		}
	}

	return &LvBatis{
		Queries: queries,
		Vars:    parseVarName(queries),
	}
}
