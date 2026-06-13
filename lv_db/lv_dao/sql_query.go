package lv_dao

import (
	"errors"
	"github.com/lostvip-com/lv_framework/lv_db"
	"github.com/lostvip-com/lv_framework/lv_db/namedsql"
	"gorm.io/gorm"
	"time"
)

func CountColumnDelFlag0(db *gorm.DB, table, column, value string) (int64, error) {
	var total int64
	err := db.Table(table).Where("del_flag=0 and "+column+"=?", value).Count(&total).Error
	return total, err
}
func CountColumnAll(db *gorm.DB, table, column, value string) (int64, error) {
	var total int64
	err := db.Table(table).Where(column+"=?", value).Count(&total).Error
	return total, err
}
func ListMapNamedSql(db *gorm.DB, sql string, req any, isCamel bool) ([]map[string]any, error) {
	return namedsql.ListMap(db, sql, req, isCamel)
}
func ListMap2MapNamedSql(db *gorm.DB, limitSql string, req any, mapKey string, isCamel bool) (map[string]map[string]any, error) {
	return namedsql.ListMap2Map(db, limitSql, req, mapKey, isCamel)
}

func ListData2MapNamedSql[T any](db *gorm.DB, limitSql string, req any, mapKey string) (map[string]T, error) {
	return namedsql.ListData2Map[T](db, limitSql, req, mapKey)
}
func GetOneMapNamedSql(db *gorm.DB, sql string, req any, isCamel bool) (map[string]any, error) {
	mp, err := namedsql.GetOneRow(db, sql, req, isCamel)
	if err != nil {
		return nil, err
	}
	return mp, err
}

// ListNamedSql 通用泛型查询
func ListNamedSql[T any](db *gorm.DB, sql string, req any) ([]T, error) {
	return namedsql.ListData[T](db, sql, req)
}

func CountNamedSql(db *gorm.DB, sql string, params any) (int64, error) {
	return namedsql.Count(db, sql, params)
}

func DeleteIds(db *gorm.DB, tableName, column string, ids []int64) error {
	var total int64
	err := db.Table(tableName).Select("count(*)").Where("id in ? ", ids).Find(&total).Error
	if total == 0 {
		return errors.New("no data found ")
	}
	delSql := "delete from " + tableName + " where id in ?"
	err = db.Exec(delSql, ids).Error
	return err
}

func Transaction(db *gorm.DB, timeout time.Duration, fn func(tx *gorm.DB) error) error {
	return lv_db.Transaction(db, timeout, fn)
}
