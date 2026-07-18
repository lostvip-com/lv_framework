package lv_ram

import (
	"sync"
	"testing"
	"time"

	"github.com/patrickmn/go-cache"
)

func TestSetGetDel(t *testing.T) {
	c := NewRamCacheClient()
	if err := c.Set("k1", "v1", cache.DefaultExpiration); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	got, err := c.Get("k1")
	if err != nil || got != "v1" {
		t.Fatalf("Get want v1, got %q err %v", got, err)
	}
	if err := c.Del("k1"); err != nil {
		t.Fatalf("Del failed: %v", err)
	}
	if _, err := c.Get("k1"); err != Nil {
		t.Fatalf("Get after Del want Nil, got %v", err)
	}
}

func TestGetMissReturnsNil(t *testing.T) {
	c := NewRamCacheClient()
	if _, err := c.Get("not-exist"); err != Nil {
		t.Fatalf("Get miss want Nil, got %v", err)
	}
}

func TestHSetHGet(t *testing.T) {
	c := NewRamCacheClient()
	if err := c.HSet("h1", "f1", "v1", "f2", "v2"); err != nil {
		t.Fatalf("HSet failed: %v", err)
	}
	if v, err := c.HGet("h1", "f1"); err != nil || v != "v1" {
		t.Fatalf("HGet f1 want v1, got %q err %v", v, err)
	}
	all, err := c.HGetAll("h1")
	if err != nil || len(all) != 2 {
		t.Fatalf("HGetAll want 2 fields, got %v (len=%d) err %v", all, len(all), err)
	}
}

// TestHMSetExpire 验证 HMSet 的 duration 对顶层 hash key 生效。
// 修复前 Expire 作用于不存在的 "pk:k"，duration 失效（会话永不过期）。
func TestHMSetExpire(t *testing.T) {
	c := NewRamCacheClient()
	fields := map[string]any{"f1": "v1", "f2": "v2"}
	if err := c.HMSet("sess1", fields, 50*time.Millisecond); err != nil {
		t.Fatalf("HMSet failed: %v", err)
	}
	if v, err := c.HGet("sess1", "f1"); err != nil || v != "v1" {
		t.Fatalf("HGet immediately want v1, got %q err %v", v, err)
	}
	time.Sleep(150 * time.Millisecond)
	if _, err := c.HGet("sess1", "f1"); err == nil {
		t.Fatalf("HGet after expire want error, got nil (session did not expire)")
	}
}

// TestGetOnHashKeyReturnsError 验证 KV 与 Hash 同 key 误用时返回 error 而非 panic（comma-ok 断言）。
func TestGetOnHashKeyReturnsError(t *testing.T) {
	c := NewRamCacheClient()
	if err := c.HSet("dup", "f1", "v1"); err != nil {
		t.Fatalf("HSet failed: %v", err)
	}
	if _, err := c.Get("dup"); err == nil {
		t.Fatalf("Get on hash key want error, got nil")
	}
}

// TestConcurrentSetGet 并发读写，go test -race 验证无 data race。
func TestConcurrentSetGet(t *testing.T) {
	c := NewRamCacheClient()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = c.Set("ck", "cv", cache.DefaultExpiration)
		}()
		go func() {
			defer wg.Done()
			_, _ = c.Get("ck")
		}()
	}
	wg.Wait()
}
