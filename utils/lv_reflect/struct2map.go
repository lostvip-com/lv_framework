package lv_reflect

import (
	"fmt"
	"reflect"
	"time"
	"unsafe"

	gocache "github.com/patrickmn/go-cache"
	"github.com/spf13/cast"
)

var cacheDuration = 5 * time.Minute
var metaCache = gocache.New(cacheDuration, 2*time.Minute)

type structMeta struct {
	fields []fieldMeta
	size   uintptr
}

type fieldMeta struct {
	offset   uintptr
	name     string
	typ      reflect.Type
	isStruct bool
	meta     *structMeta
}

// ============ 泛型版本（推荐） ============

// StructToMap 泛型版本：零开销类型推断
func StructToMap[T any](obj T) map[string]any {
	var meta *structMeta
	var ptr unsafe.Pointer

	// 编译期确定类型，避免 reflect.TypeOf 开销
	typ := reflect.TypeOf(obj)

	// 处理值或指针
	if typ.Kind() == reflect.Ptr {
		if typ.Elem().Kind() != reflect.Struct {
			return nil
		}
		val := reflect.ValueOf(obj)
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
		meta = getMeta(typ.Elem())
		ptr = unsafe.Pointer(val.Addr().Pointer())
	} else if typ.Kind() == reflect.Struct {
		meta = getMeta(typ)
		ptr = unsafe.Pointer(reflect.ValueOf(&obj).Pointer())
	} else {
		return nil
	}

	return structToMap(ptr, meta)
}

// StructsToMapSlice 泛型版本：批量转换，极致性能
func StructsToMapSlice[T any](s []T) []map[string]any {
	if len(s) == 0 {
		return nil
	}

	typ := reflect.TypeOf(s[0])
	meta := getMeta(typ)

	result := make([]map[string]any, 0, len(s))

	// 直接计算切片基址，连续内存批量处理
	basePtr := unsafe.Pointer(&s[0])

	for i := range s {
		ptr := unsafe.Pointer(uintptr(basePtr) + uintptr(i)*meta.size)
		if m := structToMap(ptr, meta); len(m) > 0 {
			result = append(result, m)
		}
	}

	return result
}

// ============ any 版本（兼容旧代码/动态类型） ============

// Struct2Map any版本：兼容接口返回值等动态场景
func Struct2Map(obj any) map[string]any {
	if obj == nil {
		return nil
	}
	// 复用泛型逻辑
	return StructToMap(obj)
}

// Structs2MapSlice any版本
func Structs2MapSlice(s []any) []map[string]any {
	if len(s) == 0 {
		return nil
	}
	// 无法确定具体类型，退化为反射遍历（性能较低）
	result := make([]map[string]any, 0, len(s))
	for _, item := range s {
		if m := Struct2Map(item); len(m) > 0 {
			result = append(result, m)
		}
	}
	return result
}

// MapToStruct 泛型版本
func MapToStruct[T any](m map[string]any) (T, error) {
	var zero T
	if m == nil {
		return zero, fmt.Errorf("nil map")
	}

	typ := reflect.TypeOf(zero)
	if typ.Kind() != reflect.Struct {
		return zero, fmt.Errorf("T must be struct")
	}

	meta := getMeta(typ)
	val := reflect.New(typ).Elem()
	ptr := unsafe.Pointer(val.Addr().Pointer())

	if err := mapToStruct(m, ptr, meta); err != nil {
		return zero, err
	}

	return val.Interface().(T), nil
}

// ============ 内部实现（优化后） ============

func getMeta(typ reflect.Type) *structMeta {
	// 解指针
	origTyp := typ
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// 缓存key使用原始类型字符串，避免指针和值的重复缓存
	cacheKey := origTyp.String()
	cached, found := metaCache.Get(cacheKey)
	if found {
		return cached.(*structMeta)
	}

	if typ.Kind() != reflect.Struct {
		return nil
	}

	meta := &structMeta{
		fields: make([]fieldMeta, 0, typ.NumField()),
		size:   typ.Size(),
	}

	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !f.IsExported() {
			continue
		}

		name := f.Tag.Get("json")
		if name == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}

		fm := fieldMeta{
			offset: f.Offset,
			name:   name,
			typ:    f.Type,
		}

		// 检查嵌套
		ft := f.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct {
			fm.isStruct = true
			fm.meta = getMeta(f.Type)
		}

		meta.fields = append(meta.fields, fm)
	}

	metaCache.Set(cacheKey, meta, cacheDuration)
	return meta
}

func structToMap(ptr unsafe.Pointer, meta *structMeta) map[string]any {
	m := make(map[string]any, len(meta.fields)) // 预分配容量

	for _, f := range meta.fields {
		fieldPtr := unsafe.Pointer(uintptr(ptr) + f.offset)
		val := readValue(fieldPtr, f)

		// 自动忽略零值
		if isZero(val) {
			continue
		}

		m[f.name] = val
	}

	return m
}

func readValue(ptr unsafe.Pointer, fm fieldMeta) any {
	// 解指针
	if fm.typ.Kind() == reflect.Ptr {
		p := *(*unsafe.Pointer)(ptr)
		if p == nil {
			return nil
		}
		ptr = p
	}

	// 嵌套结构体递归
	if fm.isStruct && fm.meta != nil {
		return structToMap(ptr, fm.meta)
	}

	// 基础类型快速路径
	switch fm.typ.Kind() {
	case reflect.Int:
		return *(*int)(ptr)
	case reflect.Int8:
		return *(*int8)(ptr)
	case reflect.Int16:
		return *(*int16)(ptr)
	case reflect.Int32:
		return *(*int32)(ptr)
	case reflect.Int64:
		return *(*int64)(ptr)
	case reflect.Uint:
		return *(*uint)(ptr)
	case reflect.Uint8:
		return *(*uint8)(ptr)
	case reflect.Uint16:
		return *(*uint16)(ptr)
	case reflect.Uint32:
		return *(*uint32)(ptr)
	case reflect.Uint64:
		return *(*uint64)(ptr)
	case reflect.Float32:
		return *(*float32)(ptr)
	case reflect.Float64:
		return *(*float64)(ptr)
	case reflect.Bool:
		return *(*bool)(ptr)
	case reflect.String:
		return *(*string)(ptr)
	default:
		// 切片、map等复杂类型
		return reflect.NewAt(fm.typ, ptr).Elem().Interface()
	}
}

func isZero(val any) bool {
	if val == nil {
		return true
	}

	switch v := val.(type) {
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(v).Int() == 0
	case uint, uint8, uint16, uint32, uint64:
		return reflect.ValueOf(v).Uint() == 0
	case float32, float64:
		return reflect.ValueOf(v).Float() == 0
	case bool:
		return !v
	case string:
		return v == ""
	case map[string]any:
		return len(v) == 0
	default:
		rv := reflect.ValueOf(val)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map, reflect.Chan:
			return rv.Len() == 0
		default:
			return false
		}
	}
}

func mapToStruct(m map[string]any, ptr unsafe.Pointer, meta *structMeta) error {
	for _, f := range meta.fields {
		mapVal, exists := m[f.name]
		if !exists || mapVal == nil {
			continue
		}

		fieldPtr := unsafe.Pointer(uintptr(ptr) + f.offset)
		if err := setValue(fieldPtr, f, mapVal); err != nil {
			return fmt.Errorf("field %s: %w", f.name, err)
		}
	}
	return nil
}

func setValue(ptr unsafe.Pointer, fm fieldMeta, val any) error {
	// 处理指针：自动创建实例
	if fm.typ.Kind() == reflect.Ptr {
		elemType := fm.typ.Elem()
		newPtr := reflect.New(elemType)
		*(*unsafe.Pointer)(ptr) = unsafe.Pointer(newPtr.Pointer())
		ptr = unsafe.Pointer(newPtr.Pointer())
	}

	// 嵌套结构体
	if fm.isStruct && fm.meta != nil {
		subMap, ok := val.(map[string]any)
		if !ok {
			return fmt.Errorf("expected map for struct field")
		}
		return mapToStruct(subMap, ptr, fm.meta)
	}

	// 使用 cast 库简化类型转换
	switch fm.typ.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		*(*int64)(ptr) = cast.ToInt64(val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		*(*uint64)(ptr) = cast.ToUint64(val)
	case reflect.Float32:
		*(*float32)(ptr) = cast.ToFloat32(val)
	case reflect.Float64:
		*(*float64)(ptr) = cast.ToFloat64(val)
	case reflect.Bool:
		*(*bool)(ptr) = cast.ToBool(val)
	case reflect.String:
		*(*string)(ptr) = cast.ToString(val)
	default:
		// 其他类型用反射
		rv := reflect.ValueOf(val)
		reflect.NewAt(fm.typ, ptr).Elem().Set(rv.Convert(fm.typ))
	}

	return nil
}

// ============ 兼容旧代码：CopyProperties2Map ============

func CopyProperties2Map(input any, result map[string]any) error {
	if result == nil {
		return fmt.Errorf("result map is nil")
	}

	m := Struct2Map(input)
	if m == nil {
		return fmt.Errorf("failed to convert struct to map")
	}

	for k, v := range m {
		result[k] = v
	}
	return nil
}
