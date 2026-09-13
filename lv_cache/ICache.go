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
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/lostvip-com/lv_framework/lv_cache/lv_ram"
	"github.com/lostvip-com/lv_framework/lv_cache/lv_redis"
	"github.com/lostvip-com/lv_framework/lv_conf"
	"github.com/lostvip-com/lv_framework/lv_global"
	"github.com/lostvip-com/lv_framework/lv_log"
	"golang.org/x/sync/singleflight"
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

// ErrNotFound 缓存未命中哨兵（供新代码语义引用；历史 miss 判定请用 IsNotFound 统一识别）
var ErrNotFound = errors.New("cache: key not found")

var (
	cacheClient ICache
	cacheOnce   sync.Once
)

// GetCacheClient 获取缓存客户端单例（并发安全，sync.Once 避免首调竞态创建多实例）
func GetCacheClient() ICache {
	cacheOnce.Do(func() {
		var config = lv_conf.Config()
		var cacheType = config.GetVipperCfg().GetString(lv_global.KEY_CACHE_TYPE)
		if cacheType == "redis" {
			cacheClient = lv_redis.GetInstance(0)
		} else {
			cacheClient = lv_ram.GetRamCacheClient()
		}
	})
	return cacheClient
}

// IsNotFound 判断缓存错误是否为"未命中"，统一识别 RAM 的 Nil 与 Redis 的 ErrNil。
// 用本函数替代业务侧 `if err != nil` 粗判 miss，可避免把网络故障误当未命中穿透到 DB。
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, lv_ram.Nil) ||
		errors.Is(err, lv_redis.ErrNil) ||
		errors.Is(err, ErrNotFound)
}

// sfGroup 合并同一 key 的并发回源请求，避免缓存击穿（热点 key 失效瞬间打 DB）
var sfGroup singleflight.Group

// WithJitter 在基础 TTL 上叠加随机抖动 [0, jitter)，避免批量 key 同时过期（缓存雪崩）
func WithJitter(base, jitter time.Duration) time.Duration {
	if jitter <= 0 {
		return base
	}
	return base + time.Duration(rand.Int63n(int64(jitter)))
}

// GetOrLoad cache-aside 模式：未命中时用 singleflight 合并并发回源。
// loader 负责查 DB 返回值；ttl 为回填缓存有效期；loader 返回 error 时不回填。
// 仅对"未命中"回源，真实错误（网络/鉴权）直接返回，避免故障穿透放大。
func GetOrLoad(key string, ttl time.Duration, loader func() (string, error)) (string, error) {
	cc := GetCacheClient()
	val, err := cc.Get(key)
	if err == nil {
		return val, nil
	}
	if !IsNotFound(err) {
		return "", err
	}
	v, loadErr, _ := sfGroup.Do(key, func() (interface{}, error) {
		s, e := loader()
		if e != nil {
			return "", e
		}
		if sErr := cc.Set(key, s, ttl); sErr != nil {
			lv_log.Error("cache Set after load error:", sErr)
		}
		return s, nil
	})
	if loadErr != nil {
		return "", loadErr
	}
	return v.(string), nil
}
