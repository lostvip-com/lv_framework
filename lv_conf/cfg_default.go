package lv_conf

import (
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/lostvip-com/lv_framework/lv_global"
	"github.com/lostvip-com/lv_framework/lv_log"
	"github.com/lostvip-com/lv_framework/utils/lv_conv"
	"github.com/lostvip-com/lv_framework/utils/lv_file"
	"github.com/lostvip-com/lv_framework/utils/lv_net"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

type CfgDefault struct {
	vipperCfg         *viper.Viper
	AppName           string
	DataSourceDefault string
	proxyMap          map[string]string
	proxyEnable       bool
	cacheTpl          bool //默认不缓存模板，方便调试
	contextPath       string
	resourcesPath     string
	logLevel          string
	autoMigrate       string
	sessionTimeout    time.Duration
}

// GetAllDataSources 获取配置文件中所有配置的数据源名称
func (e *CfgDefault) GetAllDataSources() []string {
	// 使用viper实例获取所有数据源配置
	viperCfg := e.GetVipperCfg()
	dataSourceNames := make([]string, 0)
	dataSourceMap := make(map[string]bool)

	// 从配置中获取所有以 "application.datasource." 开头且包含 ".url" 的配置键
	for _, key := range viperCfg.AllKeys() {
		if strings.HasPrefix(key, "application.datasource.") && strings.Contains(key, ".url") {
			// 提取数据源名称，例如从 "application.datasource.db-sys.url" 中提取 "db-sys"
			parts := strings.Split(key, ".")
			if len(parts) >= 3 {
				dataSourceName := parts[2]
				if !dataSourceMap[dataSourceName] {
					dataSourceMap[dataSourceName] = true
					dataSourceNames = append(dataSourceNames, dataSourceName)
				}
			}
		}
	}
	return dataSourceNames
}
func (e *CfgDefault) GetSessionTimeout(defaultTimeout time.Duration) time.Duration {
	if e.sessionTimeout > 0 {
		return e.sessionTimeout
	}
	timeoutStr := e.GetValueStr(lv_global.SESSION_TIMEOUT_KEY)
	if timeoutStr == "" { // 设置一个长期的过期时间
		lv_log.Warn("No session timeout configured! default:", defaultTimeout)
		e.sessionTimeout = defaultTimeout
	} else {
		timeout, err := time.ParseDuration(timeoutStr)
		if err != nil {
			lv_log.Error("time.ParseDuration(timeout) error:", err)
			e.sessionTimeout = defaultTimeout
		} else {
			e.sessionTimeout = timeout
		}
	}
	return e.sessionTimeout
}

func (e *CfgDefault) GetDuration(key string, defaultDuration time.Duration) time.Duration {
	timeoutStr := e.GetValueStr(lv_global.SESSION_TIMEOUT_KEY)
	if timeoutStr == "" { // 设置一个长期的过期时间
		lv_log.Warn("No Duration Configured! default:", defaultDuration)
		return defaultDuration
	} else {
		timeout, err := time.ParseDuration(timeoutStr)
		if err != nil {
			lv_log.Error("time.ParseDuration(timeout) error:", key, err)
			return defaultDuration
		}
		return timeout
	}
}
func (e *CfgDefault) GetDatasourceDefault() string {
	if e.DataSourceDefault == "" {
		e.DataSourceDefault = e.GetValueStr("application.datasource.default")
	}
	return e.DataSourceDefault
}

func (e *CfgDefault) GetResourcesPath() string {
	if e.resourcesPath == "" {
		e.resourcesPath = e.GetValueStr("application.resources-path")
	}
	return e.resourcesPath
}
func (e *CfgDefault) GetUploadPath() string {
	if e.resourcesPath == "" {
		e.resourcesPath = e.GetValueStr("application.upload-path")
	}
	return e.resourcesPath
}
func (e *CfgDefault) GetTmpPath() string {
	return "tmp" //固定临时文件目录
}
func (e *CfgDefault) GetGrpcPort() string {
	return e.GetValueStr("server.grpc.port")
}
func (e *CfgDefault) GetHost() string {
	return e.GetValueStr("server.host")
}
func (e *CfgDefault) IsProxyEnabled() bool {
	return false
}

func (e *CfgDefault) GetFuncMap() template.FuncMap {
	mp := template.FuncMap{}
	return mp
}

func (e *CfgDefault) IsCacheTpl() bool {
	return e.cacheTpl
}

func (e *CfgDefault) SetCacheTpl(cache bool) {
	e.cacheTpl = cache
}

func (e *CfgDefault) GetVipperCfg() *viper.Viper {
	return e.vipperCfg
}

func (e *CfgDefault) GetValueStrDefault(key string, defaultVal string) string {
	val := e.GetValueStr(key)
	if val == "" {
		val = defaultVal
	}
	return val
}

func (e *CfgDefault) GetValueStr(key string) string {
	if e.vipperCfg == nil {
		e.LoadConf()
	}
	val := cast.ToString(e.vipperCfg.Get(key))
	if strings.HasPrefix(val, "$") { //存在动态表达式
		val = strings.TrimSpace(val)             //去空格
		val = lv_conv.SubStr(val, 2, len(val)-1) //去掉 ${}
		if strings.HasPrefix(val, "\"") {
			panic("${...} format error !!!")
		}
		index := strings.Index(val, ":") //ssz:按第一个: 分割，前半部分是占位符，后半部分是默认值
		val0 := lv_conv.SubStr(val, 0, index)
		val0 = os.Getenv(val0) //从环境变量中取值,替换
		if val0 == "" {        //未设置环境变量,使用默认值
			val = lv_conv.SubStr(val, index+1, len(val))
			val = strings.Trim(val, "\"")
		} else {
			val = val0
		}
	}
	return val
}

func (e *CfgDefault) GetBool(key string) bool {
	if e.vipperCfg == nil {
		e.LoadConf()
	}
	val := cast.ToString(e.vipperCfg.Get(key))
	val = e.parseVal(val)
	if val == "true" {
		return true
	} else {
		return false
	}
}
func (e *CfgDefault) GetInt(key string, defaultV int) int {
	if e.vipperCfg == nil {
		e.LoadConf()
	}
	val := cast.ToString(e.vipperCfg.Get(key))
	if val == "" {
		return defaultV
	}
	return cast.ToInt(e.parseVal(val))
}
func (e *CfgDefault) parseVal(val string) string {
	if strings.HasPrefix(val, "$") { //存在动态表达式
		val = strings.TrimSpace(val)             //去空格
		val = lv_conv.SubStr(val, 2, len(val)-1) //去掉 ${}
		index := strings.Index(val, ":")         //ssz:按第一个: 分割，前半部分是占位符，后半部分是默认值
		val0 := lv_conv.SubStr(val, 0, index)
		val0 = os.Getenv(val0) //从环境变量中取值,替换
		if val0 == "" {        //未设置环境变量,使用默认值
			val = lv_conv.SubStr(val, index, len(val))
		} else {
			val = val0
		}
	}
	return val
}

func (e *CfgDefault) LoadConf() {
	currPath := lv_file.GetCurrentPath()
	fmt.Println("----> current path:" + currPath)
	e.vipperCfg = viper.New()
	fileNameArr := []string{"bootstrap", "application"}
	fileExtArr := []string{"yml", "yaml"}
	for _, fileName := range fileNameArr { //优先查找bootstrap
		for _, ext := range fileExtArr { //优先查找yaml
			for _, filePath := range BaseFilePathArr { //优先查找当前目录
				exist, yamlPath := e.MergeYarm(fileName, ext, filePath)
				if exist { //找到文件，不再寻找本目录
					fmt.Println("----> yaml path:" + yamlPath)
					break
				}
			}
		}
	}
	active := e.GetAppActive()
	if active != "" {
		e.mergeActiveYarm(active, fileExtArr, BaseFilePathArr)
	}

	if e.vipperCfg.GetBool("application.proxy.enable") == true {
		e.proxyEnable = true
		e.GetProxyMap()
	} else {
		fmt.Println("!!！！！！！！！！！！！！!!! porxy feature is disabled ！！！！！！！！！！！！！！！！！！！！！！！")
		e.proxyEnable = false
	}
}

func (e *CfgDefault) mergeActiveYarm(active string, fileExtArr []string, filePathArr []string) {
	foundActive := false
	activeFile := "application-" + active
	for _, ext := range fileExtArr { //优先查找yaml
		for _, filePath := range filePathArr { //优先查找当前目录
			exist, path := e.MergeYarm(activeFile, ext, filePath)
			if exist { //找到文件，不再寻找本目录
				foundActive = true
				fmt.Println("Active File Found: " + path)
				break
			}
		}
	}
	if !foundActive { //配置了active 却未找到
		fmt.Println("Active File Not Found, application.active:" + active)
	}
}

func (e *CfgDefault) MergeYarm(fileName, fileExt, path string) (bool, string) {
	filePath := path + "/" + fileName + "." + fileExt
	if !lv_file.IsFileExist(filePath) {
		return false, filePath //不存在
	}
	e.vipperCfg.SetConfigName(fileName)
	e.vipperCfg.SetConfigType(fileExt)
	e.vipperCfg.AddConfigPath(path)
	e.vipperCfg.MergeInConfig()
	return true, filePath
}

/**
 * app port
 */
func (e *CfgDefault) GetServerPort() int {
	port := e.GetValueStr("server.port")
	if port == "" {
		port = "8080"
	}
	return cast.ToInt(port)
}

/**
 * app port
 */
func (e *CfgDefault) GetServerIP() string {
	ip := e.GetValueStr("server.ip")
	if ip == "" {
		ip = lv_net.GetLocaHost()
	}
	return ip
}

func (e *CfgDefault) GetContextPath() string {
	return e.contextPath
}

func (e *CfgDefault) SetContextPath(ctxPath string) {
	e.contextPath = ctxPath
}

func (e *CfgDefault) GetConf(key string) string {
	v := e.GetValueStr(key)
	return v
}

func (e *CfgDefault) GetAppName() string {
	if e.AppName == "" {
		e.AppName = e.GetValueStr("application.name")
	}
	return e.AppName
}
func (e *CfgDefault) GetDriver(dbName string) string {
	key := fmt.Sprintf("application.datasource.%s.driver", dbName)
	driver := e.GetValueStr(key)
	return driver
}
func (e *CfgDefault) GetDBUrl(dbName string) string {
	key := fmt.Sprintf("application.datasource.%s.url", dbName)
	url := e.GetValueStr(key)
	if url == "" {
		url = e.GetDBUrlDefault()
	}
	return url
}
func (e *CfgDefault) GetDriverDefault() string {
	return e.GetDriver(e.GetDatasourceDefault())
}
func (e *CfgDefault) GetDBUrlDefault() string {
	return e.GetDBUrl(e.GetDatasourceDefault())
}

// IsDebug todo
func (e *CfgDefault) GetLogLevel() string {
	if e.logLevel == "" {
		e.logLevel = e.GetValueStr("application.log.level")
	}
	return e.logLevel
}

func (e *CfgDefault) GetAutoMigrate() string {
	if e.autoMigrate == "" {
		e.autoMigrate = e.GetValueStr("application.datasource.auto-migrate")
	}
	return e.autoMigrate
}

func (e *CfgDefault) GetLogOutput() string {
	output := e.GetValueStr("application.log.output")
	return output
}

func (e *CfgDefault) GetAppActive() string {
	return e.GetValueStr("application.active")
}

func (e *CfgDefault) GetNacosAddrs() string {
	return e.GetValueStr("cloud.nacos.discovery.server-addr")
}

func (e *CfgDefault) GetNacosPort() int {
	port := e.vipperCfg.GetInt("cloud.nacos.discovery.port")
	if port == 0 {
		port = 8848
	}
	return port
}
func (e *CfgDefault) GetNacosNamespace() string {
	ns := e.GetValueStr("cloud.nacos.discovery.namespace")
	return ns
}
func (e *CfgDefault) GetGroupDefault() string {
	return "DEFAULT_GROUP"
}
func (e *CfgDefault) GetDataId() string {
	key := e.GetAppName() + "-" + e.GetAppActive() + ".yml"
	fmt.Println(" dataId: " + key)
	return key
}

func (e *CfgDefault) IsProxyEnable() bool {
	return e.proxyEnable
}

func (e *CfgDefault) GetProxyMap() map[string]string {
	if e.proxyEnable && e.proxyMap == nil {
		e.LoadProxyInfo()
	}
	return e.proxyMap
}

func (e *CfgDefault) LoadProxyInfo() map[string]string {
	if !e.IsProxyEnable() {
		return nil
	}
	list := e.GetVipperCfg().GetStringSlice("application.proxy.prefix")
	e.proxyMap = make(map[string]string)
	for _, v := range list {
		index := strings.Index(v, "=")
		key := lv_conv.SubStr(v, 0, index)
		hostPort := lv_conv.SubStr(v, index+1, len(v))
		e.proxyMap[key] = hostPort
	}
	e.proxyEnable = e.GetBool("application.proxy.enable")
	lv_log.Info("application.proxy:", e.proxyMap)
	return e.proxyMap
}

func (e *CfgDefault) GetPartials() []string {
	return []string{}
} //
