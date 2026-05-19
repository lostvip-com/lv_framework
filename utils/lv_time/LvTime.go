package lv_time

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lostvip-com/lv_framework/lv_log"
)

const timeLayout = "2006-01-02 15:04:05"

// 支持的时间格式列表
var supportedFormats = []string{
	timeLayout,                            // yyyy-mm-dd hh:mm:ss 格式
	time.RFC3339Nano,                      // GORM默认的RFC3339Nano格式
	time.RFC3339,                          // 支持 2025-04-25T08:04:29.000Z 格式
	"2006-01-02 15:04:05.999999999-07:00", // 带时区信息的格式
	"2006-01-02 15:04:05.999999999+08:00", // 带+08:00时区的格式
}

type LvTime struct {
	time.Time
}

// NewTime 创建 LvTime，Go 官方建议：对于小于等于 4-5 个字段的结构体（如 time.Time），传值通常更高效
func NewTime(t time.Time) LvTime {
	return LvTime{t}
}

// GetLocalTime 获取当前本地时间
func NowTime() LvTime {
	return LvTime{time.Now()}
}

// Value 实现 driver.Valuer 接口，用于数据库写入
func (t LvTime) Value() (driver.Value, error) {
	return t.Time, nil
}

// Scan 实现 sql.Scanner 接口，用于数据库读取
func (t *LvTime) Scan(value interface{}) error {
	if value == nil {
		t.Time = time.Time{}
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		t.Time = v
		return nil
	case string: // SQLite 会走到这里
		// 依次尝试各种格式
		var tt time.Time
		var err error
		for _, layout := range supportedFormats {
			tt, err = time.Parse(layout, v)
			if err == nil {
				// 解析成功，设置时间并返回
				t.Time = tt
				return nil
			}
		}

		// 所有格式都解析失败
		lv_log.Error("cannot parse %q into LvTime, tried formats: %v, last error: %w", v, supportedFormats, err)
		return fmt.Errorf("cannot parse %q into LvTime: %w", v, err)
	default:
		return fmt.Errorf("cannot scan %T into LvTime", value)
	}
}

// MarshalJSON 实现 json.Marshaler 接口，用于 JSON 序列化
func (t LvTime) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return []byte("null"), nil
	}
	// 使用 json.Marshal 自动处理引号和转义
	return json.Marshal(t.Time.Format(timeLayout))
}

// UnmarshalJSON 实现 json.Unmarshaler 接口，用于 JSON 反序列化
func (t *LvTime) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == "" {
		t.Time = time.Time{}
		return nil
	}
	// 尝试多种时间格式进行解析
	var tt time.Time
	var err error
	for _, layout := range supportedFormats {
		tt, err = time.Parse(layout, s)
		if err == nil {
			t.Time = tt
			return nil
		}
	}
	return fmt.Errorf("cannot parse time %q, tried formats: %v", s, supportedFormats)
}
