package lv_dao

import (
	"github.com/lostvip-com/lv_framework/lv_db/lv_batis"
	"github.com/lostvip-com/lv_framework/lv_db/namedsql"
	"gorm.io/gorm"
)

func GetPageByNamedSqlTag[T any](db *gorm.DB, sqlFile string, sqlTag string, req any) ([]T, int64, error) {
	ibatis := lv_batis.NewInstance(sqlFile)
	sql, err := ibatis.GetSql(sqlTag, req)
	if err != nil {
		return nil, 0, err
	}
	return namedsql.GetPage[T](db, sql, req)
}
func GetPageMapByNamedSqlTag(db *gorm.DB, sqlFile string, sqlTag string, req any, isCamel bool) ([]map[string]any, int64, error) {
	ibatis := lv_batis.NewInstance(sqlFile)
	sql, err := ibatis.GetSql(sqlTag, req)
	if err != nil {
		return nil, 0, err
	}
	return namedsql.GetPageMap(db, sql, req, isCamel)
}
func ListMapByNamedSqlTag(db *gorm.DB, sqlFile string, sqlTag string, req any, isCamel bool) ([]map[string]any, error) {
	ibatis := lv_batis.NewInstance(sqlFile)
	sql, err := ibatis.GetSql(sqlTag, req)
	if err != nil {
		return nil, err
	}
	return namedsql.ListMap(db, sql, req, isCamel)
}
func ListDataByNamedSqlTag[T any](db *gorm.DB, sqlFile string, sqlTag string, req any) ([]T, error) {
	ibatis := lv_batis.NewInstance(sqlFile)
	sql, err := ibatis.GetSql(sqlTag, req)
	if err != nil {
		return nil, err
	}
	return namedsql.ListData[T](db, sql, req)
}
func GetSqlByTag(sqlFile string, sqlTag string, req any) (string, error) {
	ibatis := lv_batis.NewInstance(sqlFile)
	return ibatis.GetSql(sqlTag, req)
}
