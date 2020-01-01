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

package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"html/template"

	"github.com/gin-gonic/gin"
	"github.com/lostvip-com/lv_framework/lv_conf"
	"github.com/lostvip-com/lv_framework/lv_log"
	"github.com/lostvip-com/lv_framework/utils/lv_err"
	"github.com/lostvip-com/lv_framework/web/gintemplate"
	"github.com/lostvip-com/lv_framework/web/middleware"
	"github.com/lostvip-com/lv_framework/web/router"
	"github.com/spf13/cast"
)

// MyHttpServer 统一支持 HTTP/HTTPS
type MyHttpServer struct {
	HttpServer *http.Server
	//grcServer
	ServerName string
}

// ListenAndServe 启动 HTTP/HTTPS，并自带优雅关闭
func (s *MyHttpServer) ListenAndServe() {
	// 打印 Banner
	s.printBanner()
	// 优雅关闭：捕获 Ctrl+C / kill -15
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		lv_log.Info("收到退出信号，开始优雅关闭 ...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.HttpServer.Shutdown(ctx); err != nil {
			lv_log.Error("Shutdown err:", err)
		}
	}()
	ssl := lv_conf.Config().GetBool("server.ssl")
	certFile := lv_conf.Config().GetValueStr("server.cert")
	keyFile := lv_conf.Config().GetValueStr("server.key")
	// 真正启动
	var err error
	if ssl {
		lv_log.Info("⛓  HTTPS Server Listen: ", s.HttpServer.Addr)
		err = s.HttpServer.ListenAndServeTLS(certFile, keyFile)
	} else {
		lv_log.Info("⛲  HTTP Server Listen: ", s.HttpServer.Addr)
		err = s.HttpServer.ListenAndServe()
	}
	if err != nil {
		lv_log.Error("服务启动失败!!!" + err.Error())
		lv_err.PrintStackTrace(err)
		panic(err)
	}
	lv_log.Info("Server exited.")
}

// ShutDown 暴露给外部手动调用
func (s *MyHttpServer) ShutDown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.HttpServer.Shutdown(ctx)
}

// printBanner 打印控制台地址
func (s *MyHttpServer) printBanner() {
	host := lv_conf.Config().GetServerIP()
	path := lv_conf.Config().GetContextPath()
	port := cast.ToString(lv_conf.Config().GetServerPort())
	ssl := lv_conf.Config().GetBool("server.ssl")
	proto := "http"
	if ssl {
		proto = "https"
	}
	fmt.Println(strings.Repeat("#", 62))
	fmt.Println("application.name: " + lv_conf.Config().GetAppName())
	cacheType := lv_conf.Config().GetValueStr("application.cache-type")
	fmt.Println("application.cache-type: " + cacheType)
	if cacheType == "redis" {
		lv_log.Debug("application.redis.host: " + lv_conf.Config().GetValueStr("application.redis.host"))
	}
	lv_log.Debug("application.datasource.default: " + lv_conf.Config().GetDBUrlDefault())
	fmt.Printf("%s://localhost:%s%s\n", proto, port, strings.ReplaceAll(path, "//", "/"))
	fmt.Printf("%s://%s:%s%s\n", proto, host, port, strings.ReplaceAll(path, "//", "/"))
	fmt.Println(strings.Repeat("#", 62))
}

// NewHttpServer 构造器：读取配置，自动区分 HTTP/HTTPS
func NewHttpServer() *MyHttpServer {
	gin.DefaultWriter = lv_log.GetLog().GetLogWriter()
	contextPath := lv_conf.Config().GetContextPath()
	port := lv_conf.Config().GetServerPort()
	httpServer := &MyHttpServer{ServerName: lv_conf.Config().GetAppName()}
	httpServer.HttpServer = &http.Server{
		Addr:    "0.0.0.0:" + cast.ToString(port),
		Handler: InitGinRouter(contextPath),
	}
	timeoutR := lv_conf.Config().GetValueStr("server.read-timeout")
	if timeoutR != "" {
		httpServer.HttpServer.ReadTimeout = cast.ToDuration(timeoutR)
	}
	timeoutW := lv_conf.Config().GetValueStr("server.write-timeout")
	if timeoutW != "" {
		httpServer.HttpServer.ReadTimeout = cast.ToDuration(timeoutW)
	}
	return httpServer
}

// InitGinRouter 保持不变
func InitGinRouter(contextPath string) *gin.Engine {
	engine := gin.Default()
	///////////////////////中间件处理start////////////////////////////////////////////////
	engine.Use(middleware.RecoverError)
	engine.Use(middleware.SetTraceId)
	engine.Use(middleware.Options)
	engine.Use(middleware.LoggerToFile())
	engine.Use(middleware.IfProxyForward())
	//////////////////////////////////////////////////////////////////////////////////
	routerBase := engine.Group(contextPath)
	tmp, _ := os.Getwd()
	staticPath := tmp + "/resources/static"
	fmt.Println("Static Path：" + staticPath)
	routerBase.StaticFS("/static", http.Dir(staticPath))
	routerBase.StaticFile("/favicon.ico", staticPath+"/favicon.ico")

	// Get template cache TTL from config
	cacheTTL := time.Duration(0)
	if ttlStr := lv_conf.Config().GetValueStr("application.template.cache-ttl"); ttlStr != "" {
		if duration, err := time.ParseDuration(ttlStr); err == nil {
			cacheTTL = duration
		}
	}

	engine.HTMLRender = gintemplate.New(gintemplate.TemplateConfig{
		Root:      "resources/template",
		Extension: ".html",
		Master:    "",
		Partials:  lv_conf.Config().GetPartials(),
		Funcs:     template.FuncMap(lv_conf.Config().GetFuncMap()),
		CacheTTL:  cacheTTL,
	})
	// 注册业务路由
	if len(router.GroupList) > 0 {
		for _, group := range router.GroupList {
			grp := routerBase.Group(group.RelativePath, group.Handlers...)
			for _, r := range group.Router {
				if r.Method == "ANY" {
					grp.Any(r.RelativePath, r.HandlerFunc...)
				} else {
					grp.Handle(r.Method, r.RelativePath, r.HandlerFunc...)
				}
			}
		}
	}
	return engine
}
