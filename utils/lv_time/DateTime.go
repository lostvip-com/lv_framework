package lv_time

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lostvip-com/lv_framework/lv_log"
)

const datetimeLayout = "2006-01-02 15:04:05"
const dateLayout = "2006-01-02"

// 支持的时间格式列表
var supportedFormats = []string{
	datetimeLayout,                        // 2006-01-02 15:04:05
	time.RFC3339Nano,                      // GORM默认的RFC3339Nano格式
	time.RFC3339,                          // 2025-04-25T08:04:29.000Z
	"2006-01-02 15:04:05.999999999-07:00", // 带时区信息
	"2006-01-02 15:04:05.999999999+08:00", // 带+08:00时区
}

// ==================== 类型定义 ====================

type DateTime struct {
	time.Time
}

// ==================== 构造函数 ====================

// NewTime 从 time.Time 创建 DateTime
func NewTime(t time.Time) DateTime {
	return DateTime{t}
}

// NowTime 获取当前本地时间
func NowTime() DateTime {
	return DateTime{time.Now()}
}

// NewFromUnix 从 Unix 时间戳（秒）创建 DateTime
func NewFromUnix(seconds int64) DateTime {
	return DateTime{time.Unix(seconds, 0)}
}

// NewFromMillis 从 Unix 时间戳（毫秒）创建 DateTime
func NewFromMillis(millis int64) DateTime {
	return DateTime{time.UnixMilli(millis)}
}

// Parse 从字符串解析时间，自动尝试多种格式
func Parse(s string) (DateTime, error) {
	for _, layout := range supportedFormats {
		if t, err := time.Parse(layout, s); err == nil {
			return DateTime{t}, nil
		}
	}
	return DateTime{}, fmt.Errorf("cannot parse time %q, tried formats: %v", s, supportedFormats)
}

// ==================== 格式化输出 ====================

// Format 自定义格式化，默认 "2006-01-02 15:04:05"
func (t DateTime) Format(layout ...string) string {
	if t.Time.IsZero() {
		return ""
	}
	f := datetimeLayout
	if len(layout) > 0 {
		f = layout[0]
	}
	return t.Time.Format(f)
}

// FormatDate 输出日期 "2006-01-02"
func (t DateTime) FormatDate() string {
	if t.Time.IsZero() {
		return ""
	}
	return t.Time.Format(dateLayout)
}

// FormatDatetime 输出日期时间 "2006-01-02 15:04:05"
func (t DateTime) FormatDatetime() string {
	if t.Time.IsZero() {
		return ""
	}
	return t.Time.Format(datetimeLayout)
}

// ==================== 时间戳 ====================

// Unix 返回 Unix 时间戳（秒）
func (t DateTime) Unix() int64 {
	return t.Time.Unix()
}

// UnixMilli 返回 Unix 时间戳（毫秒）
func (t DateTime) UnixMilli() int64 {
	return t.Time.UnixMilli()
}

// ==================== 比较 ====================

// IsZero 是否为零值
func (t DateTime) IsZero() bool {
	return t.Time.IsZero()
}

// Before 是否在 u 之前
func (t DateTime) Before(u DateTime) bool {
	return t.Time.Before(u.Time)
}

// After 是否在 u 之后
func (t DateTime) After(u DateTime) bool {
	return t.Time.After(u.Time)
}

// ==================== 数据库接口 ====================

// Value 实现 driver.Valuer 接口，用于数据库写入
func (t DateTime) Value() (driver.Value, error) {
	return t.Time, nil
}

// Scan 实现 sql.Scanner 接口，用于数据库读取
func (t *DateTime) Scan(value any) error {
	if value == nil {
		t.Time = time.Time{}
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		t.Time = v
		return nil
	case string: // SQLite 会走到这里
		for _, layout := range supportedFormats {
			tt, err := time.Parse(layout, v)
			if err == nil {
				t.Time = tt
				return nil
			}
		}
		lv_log.Errorf("cannot parse %q into DateTime, tried formats: %v", v, supportedFormats)
		return fmt.Errorf("cannot parse %q into DateTime", v)
	default:
		return fmt.Errorf("cannot scan %T into DateTime", value)
	}
}

// ==================== JSON 接口 ====================

// MarshalJSON 实现 json.Marshaler 接口
func (t DateTime) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.Time.Format(datetimeLayout))
}

// UnmarshalJSON 实现 json.Unmarshaler 接口
func (t *DateTime) UnmarshalJSON(data []byte) error {
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
	for _, layout := range supportedFormats {
		tt, err := time.Parse(layout, s)
		if err == nil {
			t.Time = tt
			return nil
		}
	}
	return fmt.Errorf("cannot parse time %q", s)
}
