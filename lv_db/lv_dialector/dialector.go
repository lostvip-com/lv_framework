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

package lv_dialector

import (
	"fmt"
	"sort"
	"strings"
	"sync"

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
	"mysql":    &MySQLParamsProvider{},
	"sqlite":   &SQLiteParamsProvider{},
	"postgres": &PostgreSQLParamsProvider{},
}

// dialectorsMu 保护 DialectorRegistry / DefaultDialectorName / DefaultParamsProvider
var dialectorsMu sync.RWMutex

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

// init 预注册内置方言，避免在读路径上 auto-register 导致并发写 map
func init() {
	RegisterDefaultDialector("mysql", func() Dialector { return &MySQLDialector{} })
	RegisterDialector("sqlite", func() Dialector { return &SQLiteDialector{} })
	RegisterDialector("postgres", func() Dialector { return &PostgreSQLDialector{} })
}

// RegisterDialector 注册数据库方言
func RegisterDialector(name string, getDialector func() Dialector) {
	dialectorsMu.Lock()
	defer dialectorsMu.Unlock()
	DialectorRegistry[name] = getDialector
}

// RegisterDefaultDialector 注册默认数据库方言
func RegisterDefaultDialector(name string, getDialector func() Dialector) {
	RegisterDialector(name, getDialector)
	dialectorsMu.Lock()
	defer dialectorsMu.Unlock()
	DefaultDialectorName = name
}

// GetDialector 根据方言名称获取已注册的方言
func GetDialector(dialectorName string) (Dialector, error) {
	// 如果没有指定方言名称，使用默认方言
	if dialectorName == "" {
		dialectorsMu.RLock()
		dialectorName = DefaultDialectorName
		dialectorsMu.RUnlock()
	}

	// 如果仍然没有方言名称，返回错误
	if dialectorName == "" {
		return nil, fmt.Errorf("no dialector specified and no default dialector registered")
	}

	// 检查方言是否已注册（只读，不再 auto-register）
	dialectorsMu.RLock()
	getDialector, exists := DialectorRegistry[dialectorName]
	dialectorsMu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("%s dialector not registered", dialectorName)
	}

	return getDialector(), nil
}

// GetDefaultParamsByDriver 按驱动名获取默认参数（线程安全）
func GetDefaultParamsByDriver(driver string) (map[string]string, bool) {
	dialectorsMu.RLock()
	defer dialectorsMu.RUnlock()
	provider, exists := DefaultParamsProvider[driver]
	if !exists {
		return nil, false
	}
	return provider.GetDefaultParams(), true
}

// IsDialectorRegistered 检查方言是否已注册
func IsDialectorRegistered(name string) bool {
	dialectorsMu.RLock()
	defer dialectorsMu.RUnlock()
	_, exists := DialectorRegistry[name]
	return exists
}

// GetRegisteredDialectors 获取所有已注册的方言名称
func GetRegisteredDialectors() []string {
	dialectorsMu.RLock()
	defer dialectorsMu.RUnlock()
	names := make([]string, 0, len(DialectorRegistry))
	for name := range DialectorRegistry {
		names = append(names, name)
	}
	return names
}

// buildOptions 将参数 map 拼接为 URL 查询串，按 key 排序保证顺序稳定
func (cfg *DbConfig) buildOptions(options map[string]string) string {
	keys := make([]string, 0, len(options))
	for k := range options {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var opts []string
	for _, k := range keys {
		opts = append(opts, fmt.Sprintf("%s=%s", k, options[k]))
	}
	return strings.Join(opts, "&")
}

// RebuildUrl 重新构建url 使用自定义的参数覆盖默认参数
func (cfg *DbConfig) RebuildUrl() string {
	arr := strings.Split(cfg.Url, "?") //先尝试解析参数
	if len(arr) > 1 {                  //存在自定义url参数
		listAll := strings.Split(arr[1], "&")
		for _, kv := range listAll {
			// 用 SplitN(=,2) 避免值中含 '=' 被截断
			arrKV := strings.SplitN(kv, "=", 2)
			if len(arrKV) > 1 { //使用自定义的url参数覆盖默认的
				cfg.Params[arrKV[0]] = arrKV[1]
			}
		}
	}
	cfg.Url = arr[0] + "?" + cfg.buildOptions(cfg.Params)
	return cfg.Url
}
