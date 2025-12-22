package lv_dialector

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Dialector 数据库方言接口 - 专注于方言特性
type Dialector interface {
	GetName() string
	NewDialector(url string) gorm.Dialector
}

// DbParamsProvider 数据库参数提供者接口 - 负责提供数据库默认参数
type DbParamsProvider interface {
	GetDefaultParams() map[string]string
}

// DefaultParamsProvider 默认参数提供者实现
var DefaultParamsProvider = map[string]DbParamsProvider{
	"mysql": &MySQLParamsProvider{},
	"sqlite": &SQLiteParamsProvider{},
}

// MySQLParamsProvider MySQL参数提供者
type MySQLParamsProvider struct{}

func (p *MySQLParamsProvider) GetDefaultParams() map[string]string {
	return map[string]string{
		"parseTime":    "true",
		"charset":      "utf8mb4",
		"loc":          "Local",
		"timeout":      "30s",
		"readTimeout":  "30s",
		"writeTimeout": "30s",
	}
}

// SQLiteParamsProvider SQLite参数提供者
type SQLiteParamsProvider struct{}

func (p *SQLiteParamsProvider) GetDefaultParams() map[string]string {
	return map[string]string{
		"cache":         "shared",
		"mode":          "rwc",
		"_busy_timeout": "10000",
		"_journal_mode": "WAL",
		"_synchronous":  "NORMAL",
	}
}

// DbConfig 通用数据库配置
type DbConfig struct {
	DriverType   string // 驱动类型 mysql/sqlite/postgres
	Url          string
	Params       map[string]string
	ShowSql      bool
	MaxIdle      int
	MaxOpen      int
	ConnTimeout  int // 连接超时时间（秒）
	ReadTimeout  int // 读取超时时间（秒）
	WriteTimeout int // 写入超时时间（秒）
	LoggerLevel  logger.LogLevel
}

var (
	// DialectorRegistry 注册的方言映射，键为方言名称，值为创建方言实例的函数
	DialectorRegistry = make(map[string]func() Dialector, 0)
	// DefaultDialectorName 默认方言名称
	DefaultDialectorName string
)

// RegisterDialector 注册数据库方言
func RegisterDialector(name string, getDialector func() Dialector) {
	DialectorRegistry[name] = getDialector
}

// RegisterDefaultDialector 注册默认数据库方言
func RegisterDefaultDialector(name string, getDialector func() Dialector) {
	RegisterDialector(name, getDialector)
	DefaultDialectorName = name
}

// GetDialector 根据方言名称获取已注册的方言
func GetDialector(dialectorName string) (Dialector, error) {
	// 如果没有指定方言名称，使用默认方言
	if dialectorName == "" {
		dialectorName = DefaultDialectorName
	}
	
	// 如果仍然没有方言名称，返回错误
	if dialectorName == "" {
		return nil, fmt.Errorf("no dialector specified and no default dialector registered")
	}
	
	// 检查方言是否已注册
	getDialector, exists := DialectorRegistry[dialectorName]
	if !exists {
		// 自动注册常见方言
		switch dialectorName {
		case "mysql":
			RegisterDialector("mysql", func() Dialector {
				return &MySQLDialector{}
			})
		case "sqlite":
			RegisterDialector("sqlite", func() Dialector {
				return &SQLiteDialector{}
			})
		default:
			return nil, fmt.Errorf("%s dialector not registered", dialectorName)
		}
		// 重新获取注册的方言
		getDialector, exists = DialectorRegistry[dialectorName]
		if !exists {
			return nil, fmt.Errorf("failed to register %s dialector", dialectorName)
		}
	}
	
	return getDialector(), nil
}

// IsDialectorRegistered 检查方言是否已注册
func IsDialectorRegistered(name string) bool {
	_, exists := DialectorRegistry[name]
	return exists
}

// GetRegisteredDialectors 获取所有已注册的方言名称
func GetRegisteredDialectors() []string {
	var names []string
	for name := range DialectorRegistry {
		names = append(names, name)
	}
	return names
}

// 新增辅助函数处理连接选项
func (cfg *DbConfig) buildOptions(options map[string]string) string {
	var opts []string
	for k, v := range options {
		opts = append(opts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(opts, "&")
}

// RebuildUrl 重新构建url 使用自定义的参数覆盖默认参数
func (cfg *DbConfig) RebuildUrl() string {
	arr := strings.Split(cfg.Url, "?") //先尝试解析参数
	if len(arr) > 1 {                  //存在自定义url参数
		listAll := strings.Split(arr[1], "&")
		for _, kv := range listAll {
			arrKV := strings.Split(kv, "=")
			if len(arrKV) > 1 { //使用自定义的url参数覆盖默认的
				cfg.Params[arrKV[0]] = arrKV[1]
			}
		}
	}
	cfg.Url = arr[0] + "?" + cfg.buildOptions(cfg.Params)
	return cfg.Url
}
