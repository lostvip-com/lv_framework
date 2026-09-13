// Package tdengine TDengine 方言子包:引入即自注册(blank-import 按需启用)。
// 连接通道为 taosWS:纯 Go WebSocket,无 CGO 依赖,部署机免装 TDengine 客户端库。
package tdengine

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/taosdata/driver-go/v3/taosWS" // 注册纯 Go WebSocket 驱动 taosWS(经 taosAdapter)
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"github.com/lostvip-com/lv_framework/lv_db/lv_dialector"
)

func init() {
	lv_dialector.RegisterDialector("tdengine", func() lv_dialector.Dialector { return &Dialector{} })
	lv_dialector.DefaultParamsProvider["tdengine"] = &ParamsProvider{}
}

// Dialector 框架方言包装(driver 名 "tdengine")
// DSN 形如 root:taosdata@ws(localhost:6041)/dbname
type Dialector struct{}

func (d *Dialector) GetName() string {
	return "tdengine"
}

func (d *Dialector) NewDialector(url string) gorm.Dialector {
	return &GormDialector{DSN: url}
}

// ParamsProvider TDengine 参数提供者(taosWs DSN 自带全部信息,无默认查询参数)
type ParamsProvider struct{}

func (p *ParamsProvider) GetDefaultParams() map[string]string {
	return map[string]string{}
}

// GormDialector gorm 方言最小实现:仅面向原生 SQL(Raw/Exec)。
// TDengine 无通用事务、无 information_schema 迁移体系 —— 跳过默认事务、禁用 AutoMigrate,
// 表结构统一由 init.sql 维护(超级表 + 子表)。
type GormDialector struct {
	DriverName string // 默认 taosWS
	DSN        string
	Conn       gorm.ConnPool // 可注入既有连接池(测试用)
}

func (d GormDialector) Name() string { return "tdengine" }

func (d GormDialector) Initialize(db *gorm.DB) (err error) {
	driverName := d.DriverName
	if driverName == "" {
		driverName = "taosWS"
	}
	if d.Conn != nil {
		db.ConnPool = d.Conn
	} else {
		db.ConnPool, err = sql.Open(driverName, d.DSN)
		if err != nil {
			return err
		}
	}
	// TDengine 不支持跨语句事务,跳过 gorm 默认事务包装
	db.Config.SkipDefaultTransaction = true
	// 注册默认回调,保证 Raw/Exec/Scan 正常执行
	callbacks.RegisterDefaultCallbacks(db, &callbacks.Config{})
	return nil
}

func (d GormDialector) Migrator(db *gorm.DB) gorm.Migrator {
	return tdMigrator{}
}

func (d GormDialector) DataTypeOf(field *schema.Field) string {
	switch field.DataType {
	case schema.Bool:
		return "BOOL"
	case schema.Int, schema.Uint:
		if field.Size <= 16 {
			return "SMALLINT"
		} else if field.Size <= 32 {
			return "INT"
		}
		return "BIGINT"
	case schema.Float:
		return "DOUBLE"
	case schema.Time:
		return "TIMESTAMP"
	case schema.String:
		if n := field.Size; n > 0 {
			return fmt.Sprintf("VARCHAR(%d)", n)
		}
		return "NCHAR(64)"
	default:
		return "NCHAR(64)"
	}
}

func (d GormDialector) DefaultValueOf(field *schema.Field) clause.Expression {
	return clause.Expr{SQL: "DEFAULT"}
}

func (d GormDialector) BindVarTo(writer clause.Writer, stmt *gorm.Statement, v interface{}) {
	writer.WriteByte('?')
}

func (d GormDialector) QuoteTo(writer clause.Writer, str string) {
	if str == "" {
		return
	}
	// TDengine 同 MySQL 采用反引号引用,内嵌反引号双写转义
	writer.WriteByte('`')
	writer.WriteString(strings.ReplaceAll(str, "`", "``"))
	writer.WriteByte('`')
}

func (d GormDialector) Explain(sqlStr string, vars ...interface{}) string {
	return logger.ExplainSQL(sqlStr, nil, `"?"`, vars...)
}

// tdMigrator 迁移桩:TDengine 表结构由 init.sql 维护,迁移操作显式报错而非 panic
type tdMigrator struct{}

func notSupported() error {
	return fmt.Errorf("tdengine 不支持 AutoMigrate/迁移操作,请使用 init.sql 维护表结构")
}

func (m tdMigrator) AutoMigrate(dst ...interface{}) error { return notSupported() }

func (m tdMigrator) CurrentDatabase() string { return "finlens" }

func (m tdMigrator) FullDataTypeOf(field *schema.Field) clause.Expr { return clause.Expr{} }

func (m tdMigrator) GetTypeAliases(databaseTypeName string) []string { return nil }

func (m tdMigrator) CreateTable(dst ...interface{}) error { return notSupported() }
func (m tdMigrator) DropTable(dst ...interface{}) error   { return notSupported() }
func (m tdMigrator) HasTable(dst interface{}) bool        { return false }
func (m tdMigrator) RenameTable(oldName, newName interface{}) error {
	return notSupported()
}

func (m tdMigrator) GetTables() (tableList []string, err error) { return nil, notSupported() }

func (m tdMigrator) TableType(dst interface{}) (gorm.TableType, error) {
	return nil, notSupported()
}

func (m tdMigrator) AddColumn(dst interface{}, field string) error   { return notSupported() }
func (m tdMigrator) DropColumn(dst interface{}, field string) error  { return notSupported() }
func (m tdMigrator) AlterColumn(dst interface{}, field string) error { return notSupported() }
func (m tdMigrator) MigrateColumn(dst interface{}, field *schema.Field, columnType gorm.ColumnType) error {
	return notSupported()
}
func (m tdMigrator) MigrateColumnUnique(dst interface{}, field *schema.Field, columnType gorm.ColumnType) error {
	return notSupported()
}
func (m tdMigrator) HasColumn(dst interface{}, field string) bool { return false }
func (m tdMigrator) RenameColumn(dst interface{}, oldName, field string) error {
	return notSupported()
}
func (m tdMigrator) ColumnTypes(dst interface{}) ([]gorm.ColumnType, error) {
	return nil, notSupported()
}

func (m tdMigrator) CreateView(name string, option gorm.ViewOption) error { return notSupported() }
func (m tdMigrator) DropView(name string) error                           { return notSupported() }

func (m tdMigrator) CreateConstraint(dst interface{}, name string) error { return notSupported() }
func (m tdMigrator) DropConstraint(dst interface{}, name string) error   { return notSupported() }
func (m tdMigrator) HasConstraint(dst interface{}, name string) bool     { return false }

func (m tdMigrator) CreateIndex(dst interface{}, name string) error { return notSupported() }
func (m tdMigrator) DropIndex(dst interface{}, name string) error   { return notSupported() }
func (m tdMigrator) HasIndex(dst interface{}, name string) bool     { return false }
func (m tdMigrator) RenameIndex(dst interface{}, oldName, newName string) error {
	return notSupported()
}
func (m tdMigrator) GetIndexes(dst interface{}) ([]gorm.Index, error) {
	return nil, notSupported()
}
