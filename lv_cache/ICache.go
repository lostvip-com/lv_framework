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

package lv_cache

import (
	"time"

	"github.com/lostvip-com/lv_framework/lv_cache/lv_ram"
	"github.com/lostvip-com/lv_framework/lv_cache/lv_redis"
	"github.com/lostvip-com/lv_framework/lv_conf"
	"github.com/lostvip-com/lv_framework/lv_global"
)

// 支持set、hashSet操作
type ICache interface {
	Set(key string, value interface{}, expiration time.Duration) error
	Get(key string) (value string, err error)
	Del(key ...string) error

	HSet(key string, values ...interface{}) error
	HMSet(key string, mp map[string]any, duration time.Duration) error
	HGet(key, field string) (string, error)
	HDel(key string, fields ...string) error
	HGetAll(key string) (map[string]string, error)
	Exists(key string) (int64, error)
	Close() error
	Expire(key string, duration time.Duration) error
	CountKeysByPattern(pattern string) (int64, error)
	GetKeysPage(pattern string, page int, pageSize int) (keys []string, total int, err error)
}

var cacheClient ICache = nil //主数据库

func GetCacheClient() ICache {
	if cacheClient == nil {
		var config = lv_conf.Config()
		var cacheType = config.GetVipperCfg().GetString(lv_global.KEY_CACHE_TYPE)
		if cacheType == "redis" {
			cacheClient = lv_redis.GetInstance(0)
		} else {
			cacheClient = lv_ram.GetRamCacheClient()
		}
	}
	return cacheClient
}
