// Package sqlite SQLite 方言子包:引入即自注册(blank-import 按需启用)。
// 注意:底层驱动为 CGO 实现,无 CGO 的交叉编译目标不要引入本包。
package sqlite

import (
	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/lostvip-com/lv_framework/lv_db/lv_dialector"
)

func init() {
	lv_dialector.RegisterDialector("sqlite", func() lv_dialector.Dialector { return &Dialector{} })
	lv_dialector.DefaultParamsProvider["sqlite"] = &ParamsProvider{}
}

// Dialector 框架方言包装(仅专注方言特性,参数见 ParamsProvider)
type Dialector struct{}

func (d *Dialector) GetName() string { return "sqlite" }

func (d *Dialector) NewDialector(url string) gorm.Dialector {
	return gormsqlite.Open(url)
}

// ParamsProvider SQLite 参数提供者
type ParamsProvider struct{}

func (p *ParamsProvider) GetDefaultParams() map[string]string {
	return map[string]string{
		"cache":         "shared",
		"mode":          "rwc",
		"_busy_timeout": "10000",
		"_journal_mode": "WAL",
		"_synchronous":  "NORMAL",
	}
}
