package lv_dao

import (
	"context"
	"errors"
	"github.com/lostvip-com/lv_framework/lv_db"
	"github.com/lostvip-com/lv_framework/lv_db/lv_batis"
	"github.com/lostvip-com/lv_framework/lv_db/namedsql"
	"github.com/lostvip-com/lv_framework/utils/lv_err"
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
func ListMapByNamedSql(db *gorm.DB, sql string, req any, isCamel bool) (*[]map[string]any, error) {
	return namedsql.ListMap(db, sql, req, isCamel)
}

func GetOneMapByNamedSql(db *gorm.DB, sql string, req any, isCamel bool) (*map[string]any, error) {
	d := lv_db.GetOrmDefault()
	mp, err := namedsql.GetOneMapByNamedSql(d, sql, req, isCamel)
	if err != nil {
		return nil, err
	}
	return mp, err
}

func ListByNamed(db *gorm.DB, sql string, req any, isCamel bool) (*[]map[string]any, error) {
	d := lv_db.GetOrmDefault()
	return namedsql.ListMap(d, sql, req, isCamel)
}

/**
 * 通用泛型查询
 */
func ListByNamedSql[T any](db *gorm.DB, sql string, req any) (*[]T, error) {
	return namedsql.ListData[T](lv_db.GetOrmDefault(), sql, req)
}

func CountByNamedSql(db *gorm.DB, sql string, params any) (int64, error) {
	return namedsql.Count(db, sql, params)
}

/**
 * 通用泛型查询
 */
func GetPageByNamedSql[T any](db *gorm.DB, sqlfile string, sqlTag string, req any) (*[]T, int64, error) {
	//解析sql
	ibatis := lv_batis.NewInstance(sqlfile)
	sql, err := ibatis.GetLimitSql(sqlTag, req)
	lv_err.HasErrAndPanic(err)
	//查询数据
	rows, err := namedsql.ListData[T](db, sql, req)
	lv_err.HasErrAndPanic(err)
	count, err := namedsql.Count(db, sql, req)
	return rows, count, err
}

func DeleteByIds(db *gorm.DB, tableName, column string, ids []int64) error {
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
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err := db.WithContext(ctx).Transaction(fn)
	return err
}
