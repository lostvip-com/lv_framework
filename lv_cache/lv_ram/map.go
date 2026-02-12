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

package lv_ram

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

var defaultTime = 24 * 60 * time.Minute

var (
	Nil                   = errors.New("cache:nil")
	KeyNull               = errors.New("key is null")
	ValueNull             = errors.New("value is null")
	FieldNull             = errors.New("field is null")
	HashSetFieldTypeError = errors.New("hash set field's type is not string")
	FieldValueNumberError = errors.New("hash set field and value number is fault")
)
var (
	rdb *RamCacheClient
)

type RamCacheClient struct {
	c *gocache.Cache
}

func GetRamCacheClient() *RamCacheClient {
	if rdb == nil {
		rdb = NewRamCacheClient()
	}
	return rdb
}

func NewRamCacheClient() *RamCacheClient {
	return &RamCacheClient{
		c: gocache.New(30*time.Minute, 5*time.Minute),
	}
}

func (rcc *RamCacheClient) Expire(key string, duration time.Duration) error {
	val, exist := rcc.c.Get(key)
	if exist {
		rcc.c.Set(key, val, duration)
		return nil
	} else {
		return errors.New(key + "不存在")
	}
}

func (rcc *RamCacheClient) Exists(key string) (int64, error) {
	_, exist := rcc.c.Get(key)
	if exist {
		return 1, nil
	} else {
		return 0, errors.New(key + "不存在")
	}
}

func (rcc *RamCacheClient) Set(key string, value interface{}, expiration time.Duration) error {
	val, err := marshalValue(value)
	if err != nil {
		return err
	}
	if err = rcc.invalidKeyValue(key, val); err != nil {
		return err
	}
	rcc.c.Set(key, val, expiration)
	return nil
}

func (rcc *RamCacheClient) Get(key string) (data string, err error) {
	value, exist := rcc.c.Get(key)
	if !exist {
		return "", Nil
	}
	data = value.(string)
	return
}

func (rcc *RamCacheClient) Del(keys ...string) error {
	for _, key := range keys {
		rcc.c.Delete(key)
	}
	return nil
}

func (rcc *RamCacheClient) getHashCache(key string) (*gocache.Cache, bool, error) {
	if err := rcc.invalidKey(key); err != nil {
		return nil, false, err
	}
	data, exist := rcc.c.Get(key)
	if !exist {
		return nil, false, nil
	}
	value := data.(*gocache.Cache)
	return value, true, nil
}

func (rcc *RamCacheClient) HSet(key string, values ...interface{}) error {
	hashCache, exist, err := rcc.getHashCache(key)
	if err != nil {
		return err
	}
	if !exist {
		hashCache = gocache.New(gocache.DefaultExpiration, gocache.DefaultExpiration)
	}
	var field, val string
	for i := 0; i < len(values); i += 2 {
		// 防止panic，此处对hash set field做类型断言
		tp := values[i]
		switch tp.(type) {
		case string:
			field = values[i].(string)
		case map[string]any:
			return rcc.HMSet(key, values[i].(map[string]any), gocache.DefaultExpiration)
		default:
			err = HashSetFieldTypeError
			return err
		}
		// hash set的field和value数目不匹配
		if i+1 >= len(values) {
			err = FieldValueNumberError
			return err
		}
		val, err = marshalValue(values[i+1])
		if err != nil {
			return err
		}
		field = fmt.Sprintf("%v:%v", key, field)
		if err = rcc.invalidFieldValue(field, val); err != nil {
			return err
		}
		hashCache.Set(field, val, gocache.DefaultExpiration)
	}
	rcc.c.Set(key, hashCache, gocache.DefaultExpiration)
	return nil
}

func (rcc *RamCacheClient) HGet(key, field string) (string, error) {
	hashCache, exist, err := rcc.getHashCache(key)
	if err != nil {
		return "", err
	}
	if !exist {
		return "", Nil
	}
	var value interface{}
	value, exist = hashCache.Get(fmt.Sprintf("%v:%v", key, field))
	if !exist {
		return "", Nil
	}
	return value.(string), nil
}

func (rcc *RamCacheClient) HDel(key string, fields ...string) error {
	hashCache, exist, err := rcc.getHashCache(key)
	if err != nil {
		return err
	}
	if !exist {
		return nil
	}
	for _, field := range fields {
		hashCache.Delete(fmt.Sprintf("%v:%v", key, field))
	}
	return nil
}

func (rcc *RamCacheClient) HGetAll(key string) (map[string]string, error) {
	hashCache, exist, err := rcc.getHashCache(key)
	if err != nil {
		return nil, err
	}
	var mp = make(map[string]string)
	if !exist {
		return make(map[string]string), nil
	}
	for k, v := range hashCache.Items() {
		mp[strings.TrimPrefix(strings.TrimPrefix(k, key), ":")] = v.Object.(string)
	}
	return mp, nil
}

func (rcc *RamCacheClient) Close() error {
	rcc.c.Flush()
	return nil
}

// ///////////////////////////////////////////////////////////////////////////////////////////////
func marshalValue(value interface{}) (string, error) {
	if value == nil {
		return "", ValueNull
	}
	switch value.(type) {
	case string:
		return value.(string), nil
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}

func (rcc *RamCacheClient) invalidKey(key string) error {
	if key == "" {
		return KeyNull
	}
	return nil
}

func (rcc *RamCacheClient) invalidField(field string) error {
	if field == "" {
		return FieldNull
	}
	return nil
}

func (rcc *RamCacheClient) invalidValue(value string) error {
	if value == "" {
		return ValueNull
	}
	return nil
}

func (rcc *RamCacheClient) invalidKeyValue(key, value string) error {
	if err := rcc.invalidKey(key); err != nil {
		return err
	}
	return rcc.invalidValue(value)
}

func (rcc *RamCacheClient) invalidFieldValue(field, value string) error {
	if err := rcc.invalidField(field); err != nil {
		return err
	}
	return rcc.invalidValue(value)
}

func (rcc *RamCacheClient) HMSet(pk string, m map[string]any, duration time.Duration) error {
	for k, v := range m {
		rcc.HSet(pk, k, v)
		rcc.Expire(pk+":"+k, duration)
	}
	return nil
}

// 支持reids的key匹配规则
func (rcc *RamCacheClient) CountKeysByPattern(pattern string) (int64, error) {
	if pattern == "" {
		return 0, KeyNull
	}

	var count int64 = 0

	// 遍历内存中的所有键
	for key := range rcc.c.Items() {
		// 检查键是否匹配模式
		if matchPattern(key, pattern) {
			count++
		}
	}

	return count, nil
}

// matchPattern 检查键是否匹配模式
func matchPattern(key, pattern string) bool {
	// 支持 * 和 ?
	return matchPatternRecursive(key, pattern)
}

// matchPatternRecursive 递归匹配模式
func matchPatternRecursive(key, pattern string) bool {
	if pattern == "" {
		return key == ""
	}

	// 检查第一个字符
	switch pattern[0] {
	case '*':
		// * 匹配任意数量的字符
		for len(key) >= 0 {
			if matchPatternRecursive(key, pattern[1:]) {
				return true
			}
			key = key[1:]
		}
		return false
	case '?':
		// ? 匹配任意单个字符
		if len(key) == 0 {
			return false
		}
		return matchPatternRecursive(key[1:], pattern[1:])
	default:
		// 普通字符匹配
		if len(key) == 0 || key[0] != pattern[0] {
			return false
		}
		return matchPatternRecursive(key[1:], pattern[1:])
	}
}

func (rcc *RamCacheClient) GetKeysPage(pattern string, page int, pageSize int) (keys []string, total int, err error) {
	if pattern == "" {
		return nil, 0, KeyNull
	}

	var allKeys []string

	// 遍历内存中的所有键，收集匹配的键
	for key := range rcc.c.Items() {
		if matchPattern(key, pattern) {
			allKeys = append(allKeys, key)
		}
	}

	total = len(allKeys)
	if total == 0 {
		return []string{}, 0, nil
	}

	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	if start >= total {
		return []string{}, total, nil
	}

	return allKeys[start:end], total, nil
}
