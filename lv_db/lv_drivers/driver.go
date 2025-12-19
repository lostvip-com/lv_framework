package lv_drivers

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Driver 数据库驱动接口
type Driver interface {
	GetName() string
	Open(dbCfg *DbConfig) gorm.Dialector
	TestConnInstance() error
	GetDefaultParams() map[string]string
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

var DriverRegistry = make(map[string]func() Driver, 0)

// RegisterDriver 注册数据库驱动
func RegisterDriver(name string, getDriver func() Driver) {
	DriverRegistry[name] = getDriver
}

// GetDriver 根据驱动类型获取已注册的驱动
func GetDriver(driverName string) (Driver, error) {
	driverFun, exists := DriverRegistry[driverName]
	if !exists {
		return nil, fmt.Errorf("%s driver not registered", driverName)
	}
	driver := driverFun()
	return driver, nil
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
