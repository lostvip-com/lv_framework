package lv_drivers

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MySQLDriver struct {
}

func (d *MySQLDriver) GetName() string {
	return "mysql"
}

func (d *MySQLDriver) TestConnInstance() error {
	// 这里可以实现数据库连接测试逻辑
	return nil
}

func (d *MySQLDriver) GetDefaultParams() map[string]string {
	return map[string]string{
		"parseTime":    "true",
		"charset":      "utf8mb4",
		"loc":          "Local",
		"timeout":      "30s",
		"readTimeout":  "30s",
		"writeTimeout": "30s",
	}
}

func (d *MySQLDriver) Open(cfg *DbConfig) gorm.Dialector {
	// 合并默认参数和自定义参数
	params := d.GetDefaultParams()
	for k, v := range cfg.Params {
		params[k] = v
	}
	cfg.Params = params

	url := cfg.RebuildUrl()
	return mysql.Open(url)
}

//
//func (e *DbConn) CloseConn() {
//	sqlDB, err := e.gormDB.DB() //dr
//	if err == nil {
//		sqlDB.Close()
//	}
//	myDbConn = nil
//	Logger.Info(" close success !")
//	return
//}
