/*
 * Copyright 2025 lostvip
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

package lv_conf

import (
	"fmt"
	"html/template"
	"time"

	"github.com/spf13/viper"
)

var iconfig IConfig

func Config() IConfig {
	return iconfig
}
func RegisterCfg(iconf IConfig) {
	fmt.Println("=============lib_framework=======RegisterConf===========")
	iconfig = iconf
}

type IConfig interface {
	GetAllDataSources() []string
	GetDatasourceDefault() string
	GetUploadPath() string //用于提供对外服务地址
	GetResourcesPath() string
	GetTmpPath() string
	GetServerPort() int
	GetServerIP() string
	GetContextPath() string
	GetAppName() string
	GetDriver(dbName string) string
	GetDBUrl(dbName string) string
	GetDBUrlDefault() string
	GetAppActive() string
	GetNacosAddrs() string
	GetNacosPort() int
	GetNacosNamespace() string
	GetGroupDefault() string

	GetDataId() string
	GetLogLevel() string
	GetVipperCfg() *viper.Viper
	GetConf(key string) string
	GetValueStr(key string) string
	GetDuration(key string, defaultDuration time.Duration) time.Duration
	GetBool(key string) bool
	GetInt(key string, defaultV int) int
	GetProxyMap() map[string]string
	IsProxyEnabled() bool
	LoadConf()
	GetFuncMap() template.FuncMap
	GetAutoMigrate() string
	GetPartials() []string
	GetGrpcPort() string
	GetHost() string
	GetSessionTimeout(defaultTimeout time.Duration) time.Duration
	InitDatabaseDialector() error
}
