package lv_dialector

import "testing"

// TestRebuildUrlKeepsQuery 验证 URL 自带查询参数时被合并去重(按 key 排序)
func TestRebuildUrlKeepsQuery(t *testing.T) {
	cfg := DbConfig{
		DriverType: "any",
		Url:        "user:pwd@tcp(host:3306)/db?charset=utf8",
		Params: map[string]string{
			"parseTime": "true",
		},
	}
	got := cfg.RebuildUrl()
	want := "user:pwd@tcp(host:3306)/db?charset=utf8&parseTime=true"
	if got != want {
		t.Fatalf("期望: %s, 实际: %s", want, got)
	}
}

// TestRebuildUrlNoParams 无任何参数时不追加空 "?"(taosWs 等自有 DSN 格式的兼容前提)
func TestRebuildUrlNoParams(t *testing.T) {
	url := "root:pwd@ws(host:6041)/db"
	cfg := DbConfig{DriverType: "any", Url: url, Params: map[string]string{}}
	if got := cfg.RebuildUrl(); got != url {
		t.Fatalf("无参数 URL 应保持原样, 期望: %s, 实际: %s", url, got)
	}
}

// TestRegistryUnregistered 未注册方言返回可读错误(核心包不预注册任何驱动)
func TestRegistryUnregistered(t *testing.T) {
	if _, err := GetDialector("no-such-driver"); err == nil {
		t.Fatal("未注册方言应返回错误")
	}
}
