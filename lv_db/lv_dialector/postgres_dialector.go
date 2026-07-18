package lv_dialector

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgreSQLDialector struct {
}

func (d *PostgreSQLDialector) GetName() string {
	return "postgres"
}

// NewDialector 创建 PostgreSQL gorm 方言实例（实现 Dialector 接口）
func (d *PostgreSQLDialector) NewDialector(url string) gorm.Dialector {
	return postgres.Open(url)
}

// TestConnInstance 连接测试（保留以兼容旧调用）
func (d *PostgreSQLDialector) TestConnInstance() error {
	return nil
}

// GetDefaultParams 默认参数（保留以兼容旧调用）
func (d *PostgreSQLDialector) GetDefaultParams() map[string]string {
	return (&PostgreSQLParamsProvider{}).GetDefaultParams()
}

// Open 基于 DbConfig 构建 gorm 方言（保留以兼容旧调用）
func (d *PostgreSQLDialector) Open(dbCfg *DbConfig) gorm.Dialector {
	// 合并默认参数和自定义参数
	params := d.GetDefaultParams()
	for k, v := range dbCfg.Params {
		params[k] = v
	}
	dbCfg.Params = params

	url := dbCfg.RebuildUrl()
	return postgres.Open(url)
}

// PostgreSQLParamsProvider PostgreSQL 参数提供者
type PostgreSQLParamsProvider struct{}

func (p *PostgreSQLParamsProvider) GetDefaultParams() map[string]string {
	return map[string]string{
		"TimeZone":        "Asia/Shanghai",
		"connect_timeout": "30",
	}
}
