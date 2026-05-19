package lv_log_slog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/lostvip-com/lv_framework/lv_conf"
	"github.com/lostvip-com/lv_framework/lv_global"
	"github.com/lostvip-com/lv_framework/utils/lv_file"
	"github.com/spf13/cast"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LvLogSlogImpl struct {
	logger     *slog.Logger
	baseWriter io.Writer
}

func InitLog(fileName string) *LvLogSlogImpl {
	impl := &LvLogSlogImpl{}

	// 1. 确保配置已加载
	if lv_conf.Config() == nil {
		cfg := new(lv_conf.CfgDefault)
		lv_conf.RegisterCfg(cfg)
	}

	// 2. 解析日志级别
	level := impl.parseLevel()

	// 3. 创建 lumberjack logger
	lumberjackLogger := impl.createLumberjack(fileName)

	// 4. 创建多输出 writer
	impl.baseWriter = impl.createMultiWriter(lumberjackLogger)

	// 5. 创建 slog handler
	opts := &slog.HandlerOptions{
		Level: level,
	}
	handler := slog.NewTextHandler(impl.baseWriter, opts)

	// 6. 创建 logger
	impl.logger = slog.New(handler)

	// 7. 设置默认 logger
	slog.SetDefault(impl.logger)

	impl.logger.Info("slog output:", lv_conf.Config().GetValueStr("application.log.output"))
	return impl
}

func (e *LvLogSlogImpl) parseLevel() slog.Level {
	level := lv_conf.Config().GetLogLevel()

	switch level {
	case "":
		return slog.LevelError
	case "debug":
		lv_global.IsDebug = true
		fmt.Println("============ debug mod ============")
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "fatal":
		return slog.LevelError // slog 没有 fatal 级别
	case "error":
		return slog.LevelError
	default:
		panic("Log level is not support: " + level)
	}
}

func (e *LvLogSlogImpl) createLumberjack(fileName string) *lumberjack.Logger {
	maxSize := lv_conf.Config().GetValueStr("application.log.max-size")
	maxBackups := lv_conf.Config().GetValueStr("application.log.max-backups")
	maxAge := lv_conf.Config().GetValueStr("application.log.max-age")

	if maxSize == "" {
		maxSize = "200"
	}
	if maxBackups == "" {
		maxBackups = "7"
	}
	if maxAge == "" {
		maxAge = "7"
	}

	logPath := lv_file.GetCurrentPath() + "/" + lv_conf.Config().GetValueStr("application.log.path")
	err := lv_file.PathCreateIfNotExist(logPath)
	if err != nil {
		panic(err)
	}

	return &lumberjack.Logger{
		Filename:   logPath + "/" + fileName,
		MaxSize:    cast.ToInt(maxSize),
		MaxBackups: cast.ToInt(maxBackups),
		MaxAge:     cast.ToInt(maxAge),
		Compress:   true,
	}
}

func (e *LvLogSlogImpl) createMultiWriter(fileLog *lumberjack.Logger) io.Writer {
	var writers []io.Writer
	output := lv_conf.Config().GetValueStr("application.log.output")

	if strings.Contains(output, "stdout") {
		writers = append(writers, os.Stdout)
	}
	if strings.Contains(output, "file") {
		writers = append(writers, fileLog)
	}
	if len(writers) == 0 {
		writers = append(writers, os.Stdout)
	}

	return io.MultiWriter(writers...)
}

func (e *LvLogSlogImpl) GetLogWriter() io.Writer {
	return e.baseWriter
}

func (e *LvLogSlogImpl) Error(args ...interface{}) {
	e.logger.LogAttrs(context.Background(), slog.LevelError, fmt.Sprint(args...))
}

func (e *LvLogSlogImpl) ErrorTraceId(traceId any, args ...interface{}) {
	logger := e.logger.With("traceId", traceId)
	logger.LogAttrs(context.Background(), slog.LevelError, fmt.Sprint(args...))
}

func (e *LvLogSlogImpl) Fatal(args ...interface{}) {
	e.logger.LogAttrs(context.Background(), slog.LevelError, fmt.Sprint(args...))
	os.Exit(1)
}

func (e *LvLogSlogImpl) FatalTraceId(traceId any, args ...interface{}) {
	logger := e.logger.With("traceId", traceId)
	logger.LogAttrs(context.Background(), slog.LevelError, fmt.Sprint(args...))
	os.Exit(1)
}

func (e *LvLogSlogImpl) Warn(args ...interface{}) {
	e.logger.LogAttrs(context.Background(), slog.LevelWarn, fmt.Sprint(args...))
}

func (e *LvLogSlogImpl) WarnTraceId(traceId any, args ...interface{}) {
	logger := e.logger.With("traceId", traceId)
	logger.LogAttrs(context.Background(), slog.LevelWarn, fmt.Sprint(args...))
}

func (e *LvLogSlogImpl) Info(args ...interface{}) {
	e.logger.LogAttrs(context.Background(), slog.LevelInfo, fmt.Sprint(args...))
}

func (e *LvLogSlogImpl) InfoTraceId(traceId any, args ...interface{}) {
	logger := e.logger.With("traceId", traceId)
	logger.LogAttrs(context.Background(), slog.LevelInfo, fmt.Sprint(args...))
}

func (e *LvLogSlogImpl) Debug(args ...interface{}) {
	e.logger.LogAttrs(context.Background(), slog.LevelDebug, fmt.Sprint(args...))
}

func (e *LvLogSlogImpl) DebugTraceId(traceId any, args ...interface{}) {
	logger := e.logger.With("traceId", traceId)
	logger.LogAttrs(context.Background(), slog.LevelDebug, fmt.Sprint(args...))
}

// ============ 格式化日志方法 ============

func (e *LvLogSlogImpl) Errorf(format string, args ...interface{}) {
	e.logger.LogAttrs(context.Background(), slog.LevelError, fmt.Sprintf(format, args...))
}

func (e *LvLogSlogImpl) Warnf(format string, args ...interface{}) {
	e.logger.LogAttrs(context.Background(), slog.LevelWarn, fmt.Sprintf(format, args...))
}

func (e *LvLogSlogImpl) Infof(format string, args ...interface{}) {
	e.logger.LogAttrs(context.Background(), slog.LevelInfo, fmt.Sprintf(format, args...))
}

func (e *LvLogSlogImpl) Debugf(format string, args ...interface{}) {
	e.logger.LogAttrs(context.Background(), slog.LevelDebug, fmt.Sprintf(format, args...))
}
