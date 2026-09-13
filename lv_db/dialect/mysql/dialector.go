// Package mysql MySQL 方言子包:引入即自注册(blank-import 按需启用),
// 框架核心不再耦合具体数据库驱动。
package mysql

import (
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/lostvip-com/lv_framework/lv_db/lv_dialector"
)

func init() {
	lv_dialector.RegisterDefaultDialector("mysql", func() lv_dialector.Dialector { return &Dialector{} })
	lv_dialector.DefaultParamsProvider["mysql"] = &ParamsProvider{}
}

// Dialector 框架方言包装(仅专注方言特性,参数见 ParamsProvider)
type Dialector struct{}

func (d *Dialector) GetName() string { return "mysql" }

func (d *Dialector) NewDialector(url string) gorm.Dialector {
	return gormmysql.Open(url)
}

// ParamsProvider MySQL 参数提供者
type ParamsProvider struct{}

func (p *ParamsProvider) GetDefaultParams() map[string]string {
	return map[string]string{
		"parseTime":    "true",
		"charset":      "utf8mb4",
		"loc":          "Local",
		"timeout":      "30s",
		"readTimeout":  "30s",
		"writeTimeout": "30s",
	}
}
