package tdengine

import (
	"strings"
	"testing"

	"github.com/lostvip-com/lv_framework/lv_db/lv_dialector"
)

// TestRegistered 验证 init() 自注册:按名可取方言、默认参数已登记
func TestRegistered(t *testing.T) {
	d, err := lv_dialector.GetDialector("tdengine")
	if err != nil {
		t.Fatalf("tdengine 方言应已自注册: %v", err)
	}
	if d.GetName() != "tdengine" {
		t.Fatalf("方言名应为 tdengine, 实际: %s", d.GetName())
	}
	params, ok := lv_dialector.GetDefaultParamsByDriver("tdengine")
	if !ok {
		t.Fatal("tdengine 应有默认参数提供者")
	}
	if len(params) != 0 {
		t.Fatalf("tdengine 默认参数应为空(taosWs DSN 自带全部信息), 实际: %v", params)
	}
}

// TestGormDialectorDSN 验证 gorm 方言名与无参 DSN 不被破坏(不追加空 "?")
func TestGormDialectorDSN(t *testing.T) {
	wrapper, err := lv_dialector.GetDialector("tdengine")
	if err != nil {
		t.Fatalf("获取方言失败: %v", err)
	}
	gd := wrapper.NewDialector("root:taosdata@ws(localhost:6041)/finlens")
	if gd.Name() != "tdengine" {
		t.Fatalf("gorm 方言名应为 tdengine, 实际: %s", gd.Name())
	}

	dsn := "root:taosdata@ws(localhost:6041)/finlens"
	cfg := lv_dialector.DbConfig{
		DriverType: "tdengine",
		Url:        dsn,
		Params:     map[string]string{},
	}
	got := cfg.RebuildUrl()
	if got != dsn {
		t.Fatalf("DSN 应保持原样, 期望: %s, 实际: %s", dsn, got)
	}
	if strings.Contains(got, "?") {
		t.Fatalf("taosWs DSN 不应含 '?', 实际: %s", got)
	}
}
