/*
 * Copyright 2019 lostvip
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package lv_db

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lostvip-com/lv_framework/lv_conf"
	"github.com/lostvip-com/lv_framework/lv_db/lv_dialector"
	"github.com/lostvip-com/lv_framework/lv_global"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// DataSource 数据源配置
type DataSource struct {
	Name            string
	Driver          string
	URL             string
	Params          map[string]string
	Lazy            bool          // 懒加载:启动期不建连,首次 GetDBE 时才初始化(适用于可选依赖,如 TDengine)
	MaxIdle         int
	MaxOpen         int
	ConnTimeout     time.Duration // 拨号超时
	ConnMaxLifetime time.Duration // 连接最大生命周期（原 ConnTimeout 被误用作此值）
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShowSQL         bool
	LoggerLevel     logger.LogLevel
}

type Engine struct {
	dataSources map[string]*DataSource
	gormMap     map[string]*gorm.DB
	onceMap     map[string]*sync.Once // 用于确保每个数据源只初始化一次
	mu          sync.RWMutex          // 保护 gormMap / onceMap / dataSources / defaultName
	defaultName string
}

var (
	instance *Engine
	once     sync.Once
)

// GetInstance 初始化数据操作引擎（单例模式）
func GetInstance() *Engine {
	once.Do(func() {
		instance = new(Engine)
		instance.gormMap = make(map[string]*gorm.DB)
		instance.dataSources = make(map[string]*DataSource)
		instance.onceMap = make(map[string]*sync.Once)
	})
	return instance
}

// RegisterDB 注册已创建好的数据库连接
func (e *Engine) RegisterDB(name string, db *gorm.DB) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.gormMap[name] = db
	// 确保onceMap中有对应的条目，防止重复初始化
	if _, ok := e.onceMap[name]; !ok {
		e.onceMap[name] = &sync.Once{}
	}
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
	result := make(map[string]*DataSource, len(e.dataSources))
	for k, v := range e.dataSources {
		result[k] = v
	}
	return result
}

// GetDBE 根据名称获取数据库连接，失败返回 error（推荐使用）
func (e *Engine) GetDBE(name string) (*gorm.DB, error) {
	// 读路径加读锁，与运行期写（懒创建 / CloseAllConnections）互斥，避免 concurrent map read/write
	e.mu.RLock()
	if db, ok := e.gormMap[name]; ok {
		e.mu.RUnlock()
		return db, nil
	}
	e.mu.RUnlock()

	// 未命中，加写锁初始化
	e.mu.Lock()
	defer e.mu.Unlock()
	// double-check，防止在加锁前已经被其他 goroutine 初始化
	if db, ok := e.gormMap[name]; ok {
		return db, nil
	}
	ds := e.createDataSourceConfig(name)
	db, err := e.CreateAndRegisterDB(ds)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// GetDB 根据名称获取数据库连接
// Deprecated: 使用 GetDBE 以获得 error 返回；本方法保留以兼容旧调用，出错时 panic 以维持旧行为
func (e *Engine) GetDB(name string) *gorm.DB {
	db, err := e.GetDBE(name)
	if err != nil {
		panic(err)
	}
	return db
}

// SetDefaultName 设置默认数据库名称
func (e *Engine) SetDefaultName(name string) {
	e.mu.Lock()
	e.defaultName = name
	e.mu.Unlock()
}

// GetDefaultE 获取默认数据库连接，失败返回 error（推荐使用）
func (e *Engine) GetDefaultE() (*gorm.DB, error) {
	e.mu.RLock()
	name := e.defaultName
	e.mu.RUnlock()
	if name == "" {
		// 需要设置默认名称时才加写锁
		e.mu.Lock()
		if e.defaultName == "" {
			e.defaultName = lv_conf.Config().GetDatasourceDefault()
		}
		name = e.defaultName
		e.mu.Unlock()
	}
	if name == "" {
		return nil, fmt.Errorf("no default datasource configured")
	}
	return e.GetDBE(name)
}

// GetDefault 获取默认数据库连接
// Deprecated: 使用 GetDefaultE；出错时 panic 兼容旧行为
func (e *Engine) GetDefault() *gorm.DB {
	db, err := e.GetDefaultE()
	if err != nil {
		panic(err)
	}
	return db
}

// GetDBDefault 获取默认数据库连接（别名方法，用于API一致性）
// Deprecated: 使用 GetDefaultE
func (e *Engine) GetDBDefault() *gorm.DB {
	return e.GetDefault()
}

// GetOrmDefaultE 获取默认ORM实例，失败返回 error（推荐使用）
func GetOrmDefaultE() (*gorm.DB, error) {
	return GetInstance().GetDefaultE()
}

// GetOrmDefault 获取默认ORM实例
// Deprecated: 使用 GetOrmDefaultE；出错时 panic 兼容旧行为（被业务代码大量调用，保留以零改动）
func GetOrmDefault() *gorm.DB {
	return GetInstance().GetDefault()
}

// ---- context 数据源路由（为多租户/读写分离演进留口）----

type dbContextKeyType struct{}

var dbContextKey = dbContextKeyType{}

// WithDB 把数据源名写入 context，返回新 context。
// 供中间件按租户/读写场景路由数据源，service 层用 FromContext(ctx) 取回。
func WithDB(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, dbContextKey, name)
}

// FromContext 从 context 取数据源；未设置或获取失败时回退默认库（保持向后兼容）。
// 推荐新代码用 FromContext(ctx) 替代 GetOrmDefault()，避免全局默认库在并发请求间串号。
func FromContext(ctx context.Context) *gorm.DB {
	if ctx != nil {
		if name, ok := ctx.Value(dbContextKey).(string); ok && name != "" {
			if db, err := GetInstance().GetDBE(name); err == nil && db != nil {
				return db
			}
		}
	}
	return GetOrmDefault()
}

// Transaction 在指定数据库上执行事务
func Transaction(db *gorm.DB, timeout time.Duration, fn func(tx *gorm.DB) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err := db.WithContext(ctx).Transaction(fn)
	return err
}

// GetDBE 根据名称获取数据库连接（便捷方法，返回 error）
func GetDBE(name string) (*gorm.DB, error) {
	return GetInstance().GetDBE(name)
}

// GetDB 根据名称获取数据库连接（便捷方法）
// Deprecated: 使用 GetDBE
func GetDB(name string) *gorm.DB {
	return GetInstance().GetDB(name)
}

// CloseAllConnections 关闭所有数据库连接
// 注意：应在停止接收新请求、等待在途请求结束后调用；否则在途请求持有的 *gorm.DB 可能变为 bad connection
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
	// 清空连接池和onceMap，确保下次获取连接时能重新初始化
	e.gormMap = make(map[string]*gorm.DB)
	e.onceMap = make(map[string]*sync.Once)
	return err
}

// CreateAndRegisterDB 根据数据源配置创建并注册GORM实例
func (e *Engine) CreateAndRegisterDB(dataSource *DataSource) (*gorm.DB, error) {
	// 调用者必须确保已经持有了锁，这里不再加锁
	// 注册数据源配置
	e.dataSources[dataSource.Name] = dataSource
	// 确保onceMap中有对应的条目
	if _, ok := e.onceMap[dataSource.Name]; !ok {
		e.onceMap[dataSource.Name] = &sync.Once{}
	}

	// 获取对应方言
	dialector, err := lv_dialector.GetDialector(dataSource.Driver)
	if err != nil {
		return nil, fmt.Errorf("找不到方言: %s, 错误: %v", dataSource.Driver, err)
	}

	// 获取默认参数
	params := make(map[string]string)
	if defParams, exists := lv_dialector.GetDefaultParamsByDriver(dataSource.Driver); exists {
		params = defParams
	}
	// 合并自定义参数
	for k, v := range dataSource.Params {
		params[k] = v
	}

	// 创建数据库配置
	dbCfg := lv_dialector.DbConfig{
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

	// 构建完整URL
	url := dbCfg.RebuildUrl()

	// 创建gorm方言实例
	gormDialector := dialector.NewDialector(url)

	// 配置GORM
	gormCfg := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true}, // 表名使用单数
		Logger:         logger.Default.LogMode(dataSource.LoggerLevel),
	}

	// 创建GORM实例
	gormDB, err := gorm.Open(gormDialector, gormCfg)
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
	sqlDB.SetConnMaxLifetime(dataSource.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %v", err)
	}

	// 注册到引擎 - 由于已经持有锁，直接操作map
	e.gormMap[dataSource.Name] = gormDB

	return gormDB, nil
}

// InitDataSources 初始化所有数据源
func (engine *Engine) InitDataSources() {
	cfg := lv_conf.Config()
	// 获取所有配置的数据源名称
	dataSourceNames := cfg.GetAllDataSources()
	// 初始化每个数据源
	for _, dsName := range dataSourceNames {
		ds := engine.createDataSourceConfig(dsName)
		if ds.Lazy {
			// 懒加载数据源:仅登记配置,首次 GetDBE 时才建连;
			// 可选依赖(如 TDengine)不可达时 MUST NOT 阻塞主服务启动
			engine.mu.Lock()
			engine.dataSources[dsName] = ds
			engine.mu.Unlock()
			fmt.Printf("数据源 [%s] 为懒加载模式,跳过启动期建连,驱动类型: %s\n", dsName, ds.Driver)
			continue
		}
		// 使用引擎的CreateAndRegisterDB方法创建并注册数据库连接
		engine.mu.Lock()
		_, err := engine.CreateAndRegisterDB(ds)
		engine.mu.Unlock()
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
	driver := cfg.GetDriver(dsName)
	// 添加默认值处理，当驱动类型为空时使用MySQL
	if driver == "" {
		driver = "mysql"
	}
	ds := &DataSource{
		Name:            dsName,
		Driver:          driver,
		URL:             cfg.GetValueStr(fmt.Sprintf("application.datasource.%s.url", dsName)),
		Params:          make(map[string]string),
		Lazy:            cfg.GetBool(fmt.Sprintf("application.datasource.%s.lazy", dsName)),
		MaxIdle:         cfg.GetInt(fmt.Sprintf("application.datasource.%s.max-idle", dsName), 10),
		MaxOpen:         cfg.GetInt(fmt.Sprintf("application.datasource.%s.max-open", dsName), 100),
		ShowSQL:         cfg.GetBool("application.datasource.show-sql"),
		ConnTimeout:     time.Duration(cfg.GetInt(fmt.Sprintf("application.datasource.%s.conn-timeout", dsName), 30)) * time.Second,
		ConnMaxLifetime: time.Duration(cfg.GetInt(fmt.Sprintf("application.datasource.%s.conn-max-lifetime", dsName), 1800)) * time.Second,
		ReadTimeout:     time.Duration(cfg.GetInt(fmt.Sprintf("application.datasource.%s.read-timeout", dsName), 30)) * time.Second,
		WriteTimeout:    time.Duration(cfg.GetInt(fmt.Sprintf("application.datasource.%s.write-timeout", dsName), 30)) * time.Second,
	}

	// 设置日志级别
	if lv_global.IsDebug {
		ds.LoggerLevel = logger.Info
	} else {
		ds.LoggerLevel = logger.Error
	}

	return ds
}

// RegisterDialector 注册数据库方言
func (e *Engine) RegisterDialector(name string, getDialector func() lv_dialector.Dialector) {
	lv_dialector.RegisterDialector(name, getDialector)
}

// RegisterDefaultDialector 注册默认数据库方言
func (e *Engine) RegisterDefaultDialector(name string, getDialector func() lv_dialector.Dialector) {
	lv_dialector.RegisterDefaultDialector(name, getDialector)
}

// IsDialectorRegistered 检查方言是否已注册
func (e *Engine) IsDialectorRegistered(name string) bool {
	return lv_dialector.IsDialectorRegistered(name)
}

// GetRegisteredDialectors 获取所有已注册的方言名称
func (e *Engine) GetRegisteredDialectors() []string {
	return lv_dialector.GetRegisteredDialectors()
}

// ShutdownDatabase 关闭所有数据库连接
func ShutdownDatabase() {
	engine := GetInstance()
	engine.CloseAllConnections()
	fmt.Println("所有数据库连接已关闭")
}
