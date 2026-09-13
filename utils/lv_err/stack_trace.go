package lv_err

import (
	"fmt"
	"runtime/debug"

	"github.com/lostvip-com/lv_framework/lv_log"
)

// PrintStackTrace 打印 error 及其堆栈到日志（不再直写 stdout），并返回该消息。
func PrintStackTrace(err error) string {
	msg := fmt.Sprintf("err=%v, stack=%s", err, debug.Stack())
	lv_log.Error(msg)
	return msg
}
