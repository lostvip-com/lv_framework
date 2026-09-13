package lv_try

import (
	"fmt"
	"runtime/debug"

	"github.com/lostvip-com/lv_framework/lv_log"
)

// Catch 捕获 f 中的 panic 并转为 error 返回，同时记录堆栈到日志（避免静默吞错）。
func Catch(f func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			lv_log.Error(fmt.Sprintf("Recovered from panic: %v\nstack:\n%s", r, debug.Stack()))
			err = fmt.Errorf("Recovered from panic: %v", r)
		}
	}()

	// Call the provided function.
	f()

	// If we reach here, no panic occurred, so return nil error.
	return nil
}
