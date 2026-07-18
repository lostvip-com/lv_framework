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

package lv_log

import (
	"fmt"
	"io"
	"os"
)

type ILog interface {
	Error(args ...interface{})
	ErrorTraceId(traceId any, args ...interface{})
	Fatal(args ...interface{})
	FatalTraceId(traceId any, args ...interface{})
	Warn(args ...interface{})
	WarnTraceId(traceId any, args ...interface{})
	Info(args ...interface{})
	InfoTraceId(traceId any, args ...interface{})
	Debug(args ...interface{})
	DebugTraceId(traceId any, args ...interface{})
	Errorf(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	GetLogWriter() io.Writer
}

var Log ILog

// noopLogger 空实现，确保 GetLog() 在 Log 未初始化时不返回 nil（避免调用方 NPE）
type noopLogger struct{}

func (noopLogger) Error(args ...interface{})                     {}
func (noopLogger) ErrorTraceId(traceId any, args ...interface{}) {}
func (noopLogger) Fatal(args ...interface{})                     {}
func (noopLogger) FatalTraceId(traceId any, args ...interface{}) {}
func (noopLogger) Warn(args ...interface{})                      {}
func (noopLogger) WarnTraceId(traceId any, args ...interface{})  {}
func (noopLogger) Info(args ...interface{})                      {}
func (noopLogger) InfoTraceId(traceId any, args ...interface{})  {}
func (noopLogger) Debug(args ...interface{})                     {}
func (noopLogger) DebugTraceId(traceId any, args ...interface{}) {}
func (noopLogger) Errorf(format string, args ...interface{})     {}
func (noopLogger) Warnf(format string, args ...interface{})      {}
func (noopLogger) Infof(format string, args ...interface{})      {}
func (noopLogger) Debugf(format string, args ...interface{})     {}
func (noopLogger) GetLogWriter() io.Writer                       { return io.Discard }

// GetLog 返回全局 logger；未初始化时返回 noopLogger（绝不返回 nil）
func GetLog() ILog {
	if Log == nil {
		return noopLogger{}
	}
	return Log
}

func Error(args ...interface{}) {
	if Log != nil {
		Log.Error(args...)
	} else {
		fmt.Println(args...)
	}
}

func ErrorTraceId(traceId any, args ...interface{}) {
	if Log != nil {
		Log.ErrorTraceId(traceId, args...)
	} else {
		fmt.Println(args...)
	}
}

func Fatal(args ...interface{}) {
	if Log != nil {
		Log.Fatal(args...)
	} else {
		fmt.Println(args...)
		os.Exit(1)
	}
}

func FatalTraceId(traceId any, args ...interface{}) {
	if Log != nil {
		Log.FatalTraceId(traceId, args...)
	} else {
		fmt.Println(args...)
		os.Exit(1)
	}
}

func Warn(args ...interface{}) {
	if Log != nil {
		Log.Warn(args...)
	} else {
		fmt.Println(args...)
	}
}

func WarnTraceId(traceId any, args ...interface{}) {
	if Log != nil {
		Log.WarnTraceId(traceId, args...)
	} else {
		fmt.Println(args...)
	}
}

func Info(args ...interface{}) {
	if Log != nil {
		Log.Info(args...)
	} else {
		fmt.Println(args...)
	}
}

func InfoTraceId(traceId any, args ...interface{}) {
	if Log != nil {
		Log.InfoTraceId(traceId, args...)
	} else {
		fmt.Println(args...)
	}
}

func Debug(args ...interface{}) {
	if Log != nil {
		Log.Debug(args...)
	} else {
		fmt.Println(args...)
	}
}

func DebugTraceId(traceId any, args ...interface{}) {
	if Log != nil {
		Log.DebugTraceId(traceId, args...)
	} else {
		fmt.Println(args...)
	}
}

func Errorf(format string, args ...interface{}) {
	if Log != nil {
		Log.Errorf(format, args...)
	} else {
		fmt.Printf("\n"+format, args...)
	}
}

func Warnf(format string, args ...interface{}) {
	if Log != nil {
		Log.Warnf(format, args...)
	} else {
		fmt.Printf("\n"+format, args...)
	}
}

func Infof(format string, args ...interface{}) {
	if Log != nil {
		Log.Infof(format, args...)
	} else {
		fmt.Printf("\n"+format, args...)
	}
}

func Debugf(format string, args ...interface{}) {
	if Log != nil {
		Log.Debugf(format, args...)
	} else {
		fmt.Printf("\n"+format, args...)
	}
}
