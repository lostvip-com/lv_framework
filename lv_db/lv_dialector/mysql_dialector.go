package lv_dialector

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MySQLDialector struct {
}

func (d *MySQLDialector) GetName() string {
	return "mysql"
}

// MySQLDialector 仅专注于方言特性，不再负责参数配置


func (d *MySQLDialector) NewDialector(url string) gorm.Dialector {
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
