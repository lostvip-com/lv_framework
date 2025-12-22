package lv_dialector

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type SQLiteDialector struct {
}

func (d *SQLiteDialector) GetName() string {
	return "sqlite"
}

// SQLiteDialector 仅专注于方言特性，不再负责参数配置


func (d *SQLiteDialector) NewDialector(url string) gorm.Dialector {
	return sqlite.Open(url)
}
