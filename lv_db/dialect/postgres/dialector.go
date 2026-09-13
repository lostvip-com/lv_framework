// Package postgres PostgreSQL 方言子包:引入即自注册(blank-import 按需启用)。
package postgres

import (
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/lostvip-com/lv_framework/lv_db/lv_dialector"
)

func init() {
	lv_dialector.RegisterDialector("postgres", func() lv_dialector.Dialector { return &Dialector{} })
	lv_dialector.DefaultParamsProvider["postgres"] = &ParamsProvider{}
}

// Dialector 框架方言包装(仅专注方言特性,参数见 ParamsProvider)
type Dialector struct{}

func (d *Dialector) GetName() string { return "postgres" }

// NewDialector 创建 PostgreSQL gorm 方言实例（实现 Dialector 接口）
func (d *Dialector) NewDialector(url string) gorm.Dialector {
	return gormpostgres.Open(url)
}

// TestConnInstance 连接测试（保留以兼容旧调用）
func (d *Dialector) TestConnInstance() error {
	return nil
}

// GetDefaultParams 默认参数（保留以兼容旧调用）
func (d *Dialector) GetDefaultParams() map[string]string {
	return (&ParamsProvider{}).GetDefaultParams()
}

// Open 基于 DbConfig 构建 gorm 方言（保留以兼容旧调用）
func (d *Dialector) Open(dbCfg *lv_dialector.DbConfig) gorm.Dialector {
	// 合并默认参数和自定义参数
	params := d.GetDefaultParams()
	for k, v := range dbCfg.Params {
		params[k] = v
	}
	dbCfg.Params = params

	url := dbCfg.RebuildUrl()
	return gormpostgres.Open(url)
}

// ParamsProvider PostgreSQL 参数提供者
type ParamsProvider struct{}

func (p *ParamsProvider) GetDefaultParams() map[string]string {
	return map[string]string{
		"TimeZone":        "Asia/Shanghai",
		"connect_timeout": "30",
	}
}
