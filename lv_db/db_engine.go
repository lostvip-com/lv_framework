package lv_db

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lostvip-com/lv_framework/lv_conf"
	"github.com/lostvip-com/lv_framework/lv_db/lv_drivers"
	"github.com/lostvip-com/lv_framework/lv_global"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// DataSource 数据源配置
type DataSource struct {
	Name         string
	Driver       string
	URL          string
	Params       map[string]string
	MaxIdle      int
	MaxOpen      int
	ConnTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	ShowSQL      bool
	LoggerLevel  logger.LogLevel
}

type Engine struct {
	dataSources map[string]*DataSource
	gormMap     map[string]*gorm.DB
	mu          sync.RWMutex // 加锁保护并发访问
	defaultName string
}

var (
	instance *Engine
	once     sync.Once
)

func init() {
	fmt.Println("-----init orm-------")
}

// GetInstance 初始化数据操作引擎（单例模式）
func GetInstance() *Engine {
	once.Do(func() {
		instance = new(Engine)
		instance.gormMap = make(map[string]*gorm.DB)
		instance.dataSources = make(map[string]*DataSource)
	})
	return instance
}

// RegisterDB 注册已创建好的数据库连接
func (e *Engine) RegisterDB(name string, db *gorm.DB) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.gormMap[name] = db
}

// GetDataSource 获取数据源配置
func (e *Engine) GetDataSource(name string) *DataSource {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.dataSources[name]
}

// GetAllDataSources 获取所有数据源配置
func (e *Engine) GetAllDataSources() map[string]*DataSource {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make(map[string]*DataSource)
	for k, v := range e.dataSources {
		result[k] = v
	}
	return result
}

// GetDB 根据名称获取数据库连接
func (e *Engine) GetDB(name string) *gorm.DB {
	e.mu.RLock()
	db := e.gormMap[name]
	e.mu.RUnlock()
	
	if db == nil {
		e.mu.Lock()
		defer e.mu.Unlock()
		// 再次检查，防止并发情况下已经被其他goroutine创建
		if db = e.gormMap[name]; db == nil {
			ds := e.createDataSourceConfig(name)
			gdb, err := e.CreateAndRegisterDB(ds)
			if err != nil {
				panic(err)
			}
			e.gormMap[name] = gdb
			db = gdb
		}
	}
	return db
}

// SetDefaultName 设置默认数据库名称
func (e *Engine) SetDefaultName(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.defaultName = name
}

// GetDefault 获取默认数据库连接
func (e *Engine) GetDefault() *gorm.DB {
	e.mu.RLock()
	if e.defaultName == "" {
		e.defaultName = lv_conf.Config().GetDatasourceDefault()
	}
	e.mu.RUnlock()
	return e.GetDB(e.defaultName)
}

// GetDBDefault 获取默认数据库连接（别名方法，用于API一致性）
func (e *Engine) GetDBDefault() *gorm.DB {
	return e.GetDefault()
}

// GetOrmDefault 获取默认ORM实例
func GetOrmDefault() *gorm.DB {
	return GetInstance().GetDefault()
}

// Transaction 在指定数据库上执行事务
func Transaction(db *gorm.DB, timeout time.Duration, fn func(tx *gorm.DB) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err := db.WithContext(ctx).Transaction(fn)
	return err
}

// GetDB 根据名称获取数据库连接（便捷方法）
func GetDB(name string) *gorm.DB {
	return GetInstance().GetDB(name)
}

// CloseAllConnections 关闭所有数据库连接
func (e *Engine) CloseAllConnections() error {
	var err error
	e.mu.Lock()
	defer e.mu.Unlock()
	for name, db := range e.gormMap {
		if db != nil {
			sqlDB, errDB := db.DB()
			if errDB == nil {
				err = sqlDB.Close()
				fmt.Printf("数据库连接已关闭: %s\n", name)
			}
			err = errDB
		}
	}
	e.gormMap = make(map[string]*gorm.DB)
	return err
}

// RefreshConnection 刷新指定数据库连接
func (e *Engine) RefreshConnection(name string, newDB *gorm.DB) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	// 关闭旧连接
	if oldDB, exists := e.gormMap[name]; exists && oldDB != nil {
		sqlDB, err := oldDB.DB()
		if err != nil {
			return err
		}
		sqlDB.Close()
	}
	// 设置新连接
	e.gormMap[name] = newDB
	return nil
}

// CreateAndRegisterDB 根据数据源配置创建并注册GORM实例
func (e *Engine) CreateAndRegisterDB(dataSource *DataSource) (*gorm.DB, error) {
	// 注册数据源配置
	e.mu.Lock()
	e.dataSources[dataSource.Name] = dataSource
	e.mu.Unlock()
	// 获取对应驱动
	driver, err := lv_drivers.GetDriver(dataSource.Driver)
	if err != nil {
		return nil, fmt.Errorf("找不到驱动类型: %s, 错误: %v", dataSource.Driver, err)
	}

	// 合并默认参数和自定义参数
	params := driver.GetDefaultParams()
	for k, v := range dataSource.Params {
		params[k] = v
	}

	// 创建数据库配置
	dbCfg := lv_drivers.DbConfig{
		DriverType:   dataSource.Driver,
		Url:          dataSource.URL,
		Params:       params,
		ShowSql:      dataSource.ShowSQL,
		MaxIdle:      dataSource.MaxIdle,
		MaxOpen:      dataSource.MaxOpen,
		ConnTimeout:  int(dataSource.ConnTimeout.Seconds()),
		ReadTimeout:  int(dataSource.ReadTimeout.Seconds()),
		WriteTimeout: int(dataSource.WriteTimeout.Seconds()),
		LoggerLevel:  dataSource.LoggerLevel,
	}

	// 打开数据库连接
	dialector := driver.Open(&dbCfg)

	// 配置GORM
	gormCfg := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true}, // 表名使用单数
		Logger:         logger.Default.LogMode(dataSource.LoggerLevel),
	}

	// 创建GORM实例
	gormDB, err := gorm.Open(dialector, gormCfg)
	if err != nil {
		return nil, fmt.Errorf("数据库连接失败: %v", err)
	}

	// 配置连接池
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层sql.DB失败: %v", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(dataSource.MaxIdle)
	sqlDB.SetMaxOpenConns(dataSource.MaxOpen)
	sqlDB.SetConnMaxLifetime(dataSource.ConnTimeout)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %v", err)
	}

	// 注册到引擎
	e.RegisterDB(dataSource.Name, gormDB)

	return gormDB, nil
}

// initDataSources 初始化所有数据源
func (engine *Engine) InitDataSources() {
	cfg := lv_conf.Config()
	// 获取所有配置的数据源名称
	dataSourceNames := cfg.GetAllDataSources()
	// 初始化每个数据源
	for _, dsName := range dataSourceNames {
		ds := engine.createDataSourceConfig(dsName)
		// 使用引擎的CreateAndRegisterDB方法创建并注册数据库连接
		_, err := engine.CreateAndRegisterDB(ds)
		if err != nil {
			panic(fmt.Sprintf("初始化数据源 [%s] 失败: %v", dsName, err))
		}
		fmt.Printf("数据源 [%s] 初始化完成，驱动类型: %s\n", ds.Name, ds.Driver)
	}

	// 设置默认数据源
	defaultDataSource := cfg.GetDatasourceDefault()
	if defaultDataSource != "" {
		engine.SetDefaultName(defaultDataSource)
		fmt.Printf("默认数据源设置为: %s\n", defaultDataSource)
	} else if len(dataSourceNames) > 0 {
		// 如果配置中没有指定默认数据源，但存在数据源，则使用第一个作为默认
		engine.SetDefaultName(dataSourceNames[0])
		fmt.Printf("默认数据源设置为: %s\n", dataSourceNames[0])
	}
}

// createDataSourceConfig 创建数据源配置
func (engine *Engine) createDataSourceConfig(dsName string) *DataSource {
	cfg := lv_conf.Config()
	ds := &DataSource{
		Name:         dsName,
		Driver:       cfg.GetDriver(dsName),
		URL:          cfg.GetValueStr(fmt.Sprintf("application.datasource.%s.url", dsName)),
		Params:       make(map[string]string),
		MaxIdle:      cfg.GetInt(fmt.Sprintf("application.datasource.%s.max-idle", dsName), 10),
		MaxOpen:      cfg.GetInt(fmt.Sprintf("application.datasource.%s.max-open", dsName), 100),
		ShowSQL:      cfg.GetBool("application.datasource.show-sql"),
		ConnTimeout:  time.Duration(cfg.GetInt(fmt.Sprintf("application.datasource.%s.conn-timeout", dsName), 30)) * time.Second,
		ReadTimeout:  time.Duration(cfg.GetInt(fmt.Sprintf("application.datasource.%s.read-timeout", dsName), 30)) * time.Second,
		WriteTimeout: time.Duration(cfg.GetInt(fmt.Sprintf("application.datasource.%s.write-timeout", dsName), 30)) * time.Second,
	}

	// 设置日志级别
	if lv_global.IsDebug {
		ds.LoggerLevel = logger.Info
	} else {
		ds.LoggerLevel = logger.Error
	}

	return ds
}

func (e *Engine) RegisterDriver(name string, getDriver func() lv_drivers.Driver) {
	lv_drivers.RegisterDriver(name, getDriver)
}
func (e *Engine) IsDriverRegistered() bool {
	return len(lv_drivers.DriverRegistry) > 0
}

// ShutdownDatabase 关闭所有数据库连接
func ShutdownDatabase() {
	engine := GetInstance()
	engine.CloseAllConnections()
	fmt.Println("所有数据库连接已关闭")
}
