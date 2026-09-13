package lv_db

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lostvip-com/lv_framework/lv_conf"
	"github.com/lostvip-com/lv_framework/lv_db/lv_dialector"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// ---- 测试专用桩驱动/方言:连接恒失败,不依赖任何真实数据库驱动 ----
// (核心包已不预注册驱动,lazy 语义测试自带注册)

const stubDriverName = "lazystub"

type failingDriver struct{}

func (failingDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("lazystub: 连接失败(测试桩)")
}

type stubGormDialector struct{ dsn string }

func (d stubGormDialector) Name() string { return stubDriverName }

// ---- 框架方言语接(lv_dialector.Dialector)----

func (d stubGormDialector) GetName() string { return stubDriverName }

func (d stubGormDialector) NewDialector(url string) gorm.Dialector {
	return stubGormDialector{dsn: url}
}

func (d stubGormDialector) Initialize(db *gorm.DB) error {
	sqlDB, err := sql.Open(stubDriverName, d.dsn)
	if err != nil {
		return err
	}
	db.ConnPool = sqlDB
	return nil
}

func (d stubGormDialector) Migrator(*gorm.DB) gorm.Migrator { return nil }

func (d stubGormDialector) DataTypeOf(*schema.Field) string { return "TEXT" }

func (d stubGormDialector) DefaultValueOf(*schema.Field) clause.Expression {
	return clause.Expr{SQL: "DEFAULT"}
}

func (d stubGormDialector) BindVarTo(writer clause.Writer, _ *gorm.Statement, _ interface{}) {
	writer.WriteByte('?')
}

func (d stubGormDialector) QuoteTo(writer clause.Writer, str string) {
	writer.WriteString(str)
}

func (d stubGormDialector) Explain(sqlStr string, _ ...interface{}) string { return sqlStr }

func init() {
	sql.Register(stubDriverName, failingDriver{})
	// 桩方言自注册:核心包不再预注册任何真实驱动
	lv_dialector.RegisterDialector(stubDriverName, func() lv_dialector.Dialector {
		return stubGormDialector{}
	})
}

// stubConfig 测试用最小 IConfig 实现:仅 db-lazy-test 数据源
type stubConfig struct {
	lv_conf.CfgDefault
	lazy bool
}

func (s *stubConfig) GetAllDataSources() []string {
	return []string{"db-lazy-test"}
}

func (s *stubConfig) GetDatasourceDefault() string { return "" }

func (s *stubConfig) GetDriver(dbName string) string { return stubDriverName }

func (s *stubConfig) GetValueStr(key string) string {
	if strings.HasSuffix(key, ".url") {
		return "lazystub://unreachable"
	}
	if strings.HasSuffix(key, ".lazy") {
		if s.lazy {
			return "true"
		}
		return "false"
	}
	return ""
}

func (s *stubConfig) GetBool(key string) bool {
	return strings.HasSuffix(key, ".lazy") && s.lazy
}

func (s *stubConfig) GetInt(key string, defaultV int) int { return defaultV }

func (s *stubConfig) GetDuration(key string, defaultDuration time.Duration) time.Duration {
	return defaultDuration
}

// ---- 以下为无关方法的空实现 ----

func (s *stubConfig) GetUploadPath() string      { return "" }
func (s *stubConfig) GetResourcesPath() string   { return "" }
func (s *stubConfig) GetTmpPath() string         { return "" }
func (s *stubConfig) GetServerPort() int         { return 8080 }
func (s *stubConfig) GetServerIP() string        { return "127.0.0.1" }
func (s *stubConfig) GetContextPath() string     { return "" }
func (s *stubConfig) GetAppName() string         { return "test" }
func (s *stubConfig) GetDBUrl(dbName string) string { return "" }
func (s *stubConfig) GetDBUrlDefault() string    { return "" }
func (s *stubConfig) GetAppActive() string       { return "test" }
func (s *stubConfig) GetLogLevel() string        { return "error" }
func (s *stubConfig) GetVipperCfg() *viper.Viper { return nil }
func (s *stubConfig) GetConf(key string) string  { return "" }
func (s *stubConfig) LoadConf()                  {}
func (s *stubConfig) GetAutoMigrate() string     { return "none" }
func (s *stubConfig) GetGrpcPort() string        { return "" }
func (s *stubConfig) GetHost() string            { return "" }
func (s *stubConfig) GetSessionTimeout(defaultTimeout time.Duration) time.Duration {
	return defaultTimeout
}
func (s *stubConfig) InitDatabaseDialector() error { return nil }

// TestLazyDataSourceNotBlockingStartup 懒加载源不可达时:
// InitDataSources 不 panic、配置仅登记、GetDBE 返回可读错误
func TestLazyDataSourceNotBlockingStartup(t *testing.T) {
	lv_conf.RegisterCfg(&stubConfig{lazy: true})
	engine := GetInstance()
	engine.SetDefaultName("") // 清理上一次测试可能留下的默认源

	engine.InitDataSources() // MUST NOT panic(数据源不可达)

	ds := engine.GetDataSource("db-lazy-test")
	if ds == nil {
		t.Fatal("懒加载源应仅登记配置")
	}
	if !ds.Lazy {
		t.Fatal("Lazy 标志应被读取为 true")
	}
	if _, err := engine.GetDBE("db-lazy-test"); err == nil {
		t.Fatal("不可达的懒加载源 GetDBE 应返回可读错误")
	}
	// 恢复默认源,避免污染其他测试
	engine.SetDefaultName("")
}

// TestNonLazyStillFailsFast 非懒数据源不可达时保持启动快速失败(panic)语义
func TestNonLazyStillFailsFast(t *testing.T) {
	lv_conf.RegisterCfg(&stubConfig{lazy: false})
	engine := GetInstance()
	engine.SetDefaultName("")

	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		engine.InitDataSources()
	}()
	if !panicked {
		t.Fatal("非懒数据源不可达时 InitDataSources 应快速失败(panic)")
	}
	engine.SetDefaultName("")
}
