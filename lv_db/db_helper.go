package lv_db

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// DBHelper 数据库助手工具类，提供便捷的数据库操作接口
type DBHelper struct {
	engine *Engine
}

// NewDBHelper 创建数据库助手实例
func NewDBHelper() *DBHelper {
	return &DBHelper{
		engine: GetInstance(),
	}
}

// GetDB 获取指定名称的数据库连接
func (h *DBHelper) GetDB(dbName string) *gorm.DB {
	if dbName == "" {
		return h.engine.GetDBDefault()
	}
	return h.engine.GetDB(dbName)
}

// GetDefaultDB 获取默认数据库连接
func (h *DBHelper) GetDefaultDB() *gorm.DB {
	return h.engine.GetDBDefault()
}

// SwitchDB 切换数据库连接
func (h *DBHelper) SwitchDB(dbName string) *gorm.DB {
	db := h.GetDB(dbName)
	if db == nil {
		panic(fmt.Sprintf("数据库 [%s] 不存在", dbName))
	}
	return db
}

// TestConnection 测试数据库连接
func (h *DBHelper) TestConnection(dbName string) error {
	db := h.GetDB(dbName)
	if db == nil {
		return fmt.Errorf("数据库 [%s] 不存在", dbName)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// RegisterNewDataSource 动态注册新的数据源
func (h *DBHelper) RegisterNewDataSource(dataSource *DataSource) error {
	// 直接调用引擎的CreateAndRegisterDB方法，该方法会自动注册数据源并创建GORM实例
	_, err := h.engine.CreateAndRegisterDB(dataSource)
	if err != nil {
		return fmt.Errorf("注册数据源失败: %v", err)
	}
	return nil
}

// CloseAllConnections 关闭所有数据库连接
func (h *DBHelper) CloseAllConnections() error {
	return h.engine.CloseAllConnections()
}

// GetDataSource 获取数据源配置
func (h *DBHelper) GetDataSource(dbName string) *DataSource {
	return h.engine.GetDataSource(dbName)
}

// IsTransaction 检查当前是否在事务中
func (h *DBHelper) IsTransaction(db *gorm.DB) bool {
	return db.Statement.ConnPool != nil && db.Error == nil
}

// ExecuteInTransaction 在事务中执行操作
func (h *DBHelper) ExecuteInTransaction(dbName string, fn func(*gorm.DB) error) error {
	db := h.GetDB(dbName)
	return db.Transaction(fn)
}

// GetDriverType 获取数据库驱动类型
func (h *DBHelper) GetDriverType(dbName string) string {
	ds := h.GetDataSource(dbName)
	if ds != nil {
		return ds.Driver
	}
	return ""
}

// IsMySQL 检查是否为MySQL数据库
func (h *DBHelper) IsMySQL(dbName string) bool {
	return strings.EqualFold(h.GetDriverType(dbName), "mysql")
}

// IsSQLite 检查是否为SQLite数据库
func (h *DBHelper) IsSQLite(dbName string) bool {
	return strings.EqualFold(h.GetDriverType(dbName), "sqlite")
}

// IsPostgreSQL 检查是否为PostgreSQL数据库
func (h *DBHelper) IsPostgreSQL(dbName string) bool {
	return strings.EqualFold(h.GetDriverType(dbName), "postgres")
}

// GetDBVersion 获取数据库版本信息
func (h *DBHelper) GetDBVersion(dbName string) (string, error) {
	db := h.GetDB(dbName)
	if db == nil {
		return "", fmt.Errorf("数据库 [%s] 不存在", dbName)
	}

	var version string
	switch {
	case h.IsMySQL(dbName):
		result := db.Raw("SELECT VERSION()").Scan(&version)
		return version, result.Error
	case h.IsPostgreSQL(dbName):
		result := db.Raw("SELECT version()").Scan(&version)
		return version, result.Error
	case h.IsSQLite(dbName):
		result := db.Raw("SELECT sqlite_version()").Scan(&version)
		return version, result.Error
	default:
		return "未知数据库类型", nil
	}
}

// 全局数据库助手实例
var globalDBHelper *DBHelper

// GetDBHelper 获取全局数据库助手实例
func GetDBHelper() *DBHelper {
	if globalDBHelper == nil {
		globalDBHelper = NewDBHelper()
	}
	return globalDBHelper
}
