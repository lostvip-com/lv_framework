package lv_db

import (
	"sync"
	"testing"

	"gorm.io/gorm"
)

// TestConcurrentGetDBRegistered 验证已注册数据源的并发 GetDB 快速路径不再触发
// "concurrent map read and map write"（修复前 GetDB 快速路径无锁读 gormMap）。
// 推荐以 go test -race ./lv_db/ 运行（需 cgo）。
func TestConcurrentGetDBRegistered(t *testing.T) {
	engine := GetInstance()
	// 注册一个不连真实DB的 gorm.DB，仅用于测试 map 并发读路径
	engine.RegisterDB("test-concurrent", &gorm.DB{})

	concurrency := 100
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			db := GetDB("test-concurrent") // 已注册，走快速路径 RLock 读
			if db == nil {
				t.Error("GetDB returned nil")
			}
		}()
	}
	wg.Wait()
}
