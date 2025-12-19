package lv_drivers

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type SQLiteDriver struct {
}

func (d *SQLiteDriver) GetName() string {
	return "sqlite"
}

func (d *SQLiteDriver) TestConnInstance() error {
	// 这里可以实现数据库连接测试逻辑
	return nil
}

func (d *SQLiteDriver) GetDefaultParams() map[string]string {
	return map[string]string{
		"cache":             "shared",
		"mode":              "rwc",
		"_busy_timeout":     "10000",
		"_journal_mode":     "WAL",
		"_synchronous":      "NORMAL",
	}
}

func (d *SQLiteDriver) Open(dbCfg *DbConfig) gorm.Dialector {
	// 合并默认参数和自定义参数
	params := d.GetDefaultParams()
	for k, v := range dbCfg.Params {
		params[k] = v
	}
	dbCfg.Params = params
	
	url := dbCfg.RebuildUrl()
	return sqlite.Open(url)
}
