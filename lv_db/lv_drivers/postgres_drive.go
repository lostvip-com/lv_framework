package lv_drivers

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgreSQLDriver struct {
}

func (d *PostgreSQLDriver) GetName() string {
	return "postgres"
}

func (d *PostgreSQLDriver) TestConnInstance() error {
	// 这里可以实现数据库连接测试逻辑
	return nil
}

func (d *PostgreSQLDriver) GetDefaultParams() map[string]string {
	return map[string]string{
		"TimeZone":        "Asia/Shanghai",
		"connect_timeout": "30",
	}
}

func (d *PostgreSQLDriver) Open(dbCfg *DbConfig) gorm.Dialector {
	// 合并默认参数和自定义参数
	params := d.GetDefaultParams()
	for k, v := range dbCfg.Params {
		params[k] = v
	}
	dbCfg.Params = params

	url := dbCfg.RebuildUrl()
	return postgres.Open(url)
}
