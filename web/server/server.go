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

	"github.com/gin-gonic/gin"
	"github.com/lostvip-com/lv_framework/lv_global"
	"github.com/lostvip-com/lv_framework/lv_log"
	"github.com/lostvip-com/lv_framework/utils/lv_err"
	"github.com/lostvip-com/lv_framework/web/gintemplate"
	"github.com/lostvip-com/lv_framework/web/middleware"
	"github.com/lostvip-com/lv_framework/web/router"
	"github.com/spf13/cast"
	"html/template"
)

// MyHttpServer 统一支持 HTTP/HTTPS
type MyHttpServer struct {
	server     *http.Server
	ServerName string
	Address    string
	ServerRoot string
	Handler    *gin.Engine

	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxHeaderBytes int

	CertFile string // 证书路径，空则走 HTTP
	KeyFile  string // 私钥路径
}

// ListenAndServe 启动 HTTP/HTTPS，并自带优雅关闭
func (s *MyHttpServer) ListenAndServe() {
	s.server = &http.Server{
		Addr:           s.Address,
		Handler:        s.Handler,
		ReadTimeout:    s.ReadTimeout,
		WriteTimeout:   s.WriteTimeout,
		MaxHeaderBytes: s.MaxHeaderBytes,
	}

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
		if err := s.server.Shutdown(ctx); err != nil {
			lv_log.Error("Shutdown err:", err)
		}
	}()

	// 真正启动
	var err error
	if s.CertFile != "" && s.KeyFile != "" {
		lv_log.Info("⛓  HTTPS Server Listen: ", s.Address)
		err = s.server.ListenAndServeTLS(s.CertFile, s.KeyFile)
	} else {
		lv_log.Info("⛲  HTTP Server Listen: ", s.Address)
		err = s.server.ListenAndServe()
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
	return s.server.Shutdown(ctx)
}

// printBanner 打印控制台地址
func (s *MyHttpServer) printBanner() {
	host := lv_global.Config().GetServerIP()
	path := lv_global.Config().GetContextPath()
	port := cast.ToString(lv_global.Config().GetServerPort())
	proto := "http"
	if s.CertFile != "" && s.KeyFile != "" {
		proto = "https"
	}
	fmt.Println(strings.Repeat("#", 62))
	fmt.Println("application.name: " + lv_global.Config().GetAppName())
	fmt.Println("application.cache: " + lv_global.Config().GetValueStr("application.cache"))
	fmt.Println("application.redis.host: " + lv_global.Config().GetValueStr("application.redis.host"))
	fmt.Println("application.datasource.default: " + lv_global.Config().GetDBUrlDefault())
	fmt.Printf("%s://localhost:%s%s\n", proto, port, strings.ReplaceAll(path, "//", "/"))
	fmt.Printf("%s://%s:%s%s\n", proto, host, port, strings.ReplaceAll(path, "//", "/"))
	fmt.Println(strings.Repeat("#", 62))
}

// NewHttpServer 构造器：读取配置，自动区分 HTTP/HTTPS
func NewHttpServer() *MyHttpServer {
	gin.DefaultWriter = lv_log.GetLog().GetLogWriter()
	contextPath := lv_global.Config().GetContextPath()
	port := lv_global.Config().GetServerPort()
	certFile := lv_global.Config().GetValueStr("server.cert")
	keyFile := lv_global.Config().GetValueStr("server.key")
	return &MyHttpServer{
		ServerName:     lv_global.Config().GetAppName(),
		Address:        "0.0.0.0:" + cast.ToString(port),
		Handler:        InitGinRouter(contextPath),
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
		CertFile:       certFile, // 配置空则走 HTTP
		KeyFile:        keyFile,
	}
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

	engine.HTMLRender = gintemplate.New(gintemplate.TemplateConfig{
		Root:      "resources/template",
		Extension: ".html",
		Master:    "",
		Partials:  lv_global.Config().GetPartials(),
		Funcs:     template.FuncMap(lv_global.Config().GetFuncMap()),
		CacheTpl:  lv_global.Config().IsCacheTpl(),
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
