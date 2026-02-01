/*
 * Copyright 2025 lostvip
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

package lv_redis

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lostvip-com/lv_framework/lv_conf"
	"github.com/lostvip-com/lv_framework/lv_log"
	"github.com/redis/go-redis/v9"
)

var (
	redisClient *RedisClient
	onceRedis   = sync.Once{}
)

type RedisClient struct {
	client *redis.Client
}

func GetInstance(indexDb int) *RedisClient {
	if redisClient == nil {
		onceRedis.Do(func() {
			redisClient = NewRedisClient(indexDb)
		})
	}
	return redisClient
}

func NewRedisClient(indexDb int) *RedisClient {
	conf := lv_conf.CfgDefault{}
	addr := conf.GetValueStr("application.redis.host")
	port := conf.GetValueStr("application.redis.port")
	password := conf.GetValueStr("application.redis.password")
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr + ":" + port,
		Password: password, // 没有密码，默认值
		DB:       indexDb,  // 默认DB 0
	})

	redisClient := &RedisClient{client: rdb}

	// 测试连接
	if _, err := redisClient.Ping(context.Background()); err != nil {
		msg := `
              ------------>连接 Redis 错误：
              无法链接到 Redis!!!! 检查相关配置:
              host: %v
              port: %v
              password: %v
             `
		lv_log.Error(fmt.Sprintf(msg, addr, port, password))
		panic(fmt.Sprintf("Redis 错误: host: %v port: %v", addr, port))
	}

	return redisClient
}

func (rcc *RedisClient) Ping(ctx context.Context) (string, error) {
	return rcc.client.Ping(ctx).Result()
}

func (rcc *RedisClient) HMSet(key string, mp map[string]any, expiration time.Duration) error {
	err := rcc.client.HSet(context.Background(), key, mp).Err()
	if err != nil {
		lv_log.Error(fmt.Sprintf("HMSet error: %v", err))
		return err
	}
	if expiration > 0 {
		if err := rcc.client.Expire(context.Background(), key, expiration).Err(); err != nil {
			lv_log.Error(fmt.Sprintf("Expire error: %v", err))
			return err
		}
	}
	return nil
}

func (rcc *RedisClient) Expire(key string, duration time.Duration) error {
	return rcc.client.Expire(context.Background(), key, duration).Err()
}

func (rcc *RedisClient) Exists(key string) (int64, error) {
	return rcc.client.Exists(context.Background(), key).Result()
}

func (rcc *RedisClient) Set(key string, value interface{}, expiration time.Duration) error {
	if err := rcc.client.Set(context.Background(), key, value, expiration).Err(); err != nil {
		lv_log.Error(fmt.Sprintf("Set error: %v", err))
		return err
	}
	return nil
}

func (rcc *RedisClient) Get(key string) (string, error) {
	data, err := rcc.client.Get(context.Background(), key).Result()
	if err != nil {
		lv_log.Error(fmt.Sprintf("Get error: %v", err))
		return "", err
	}
	return data, nil
}

func (rcc *RedisClient) Del(keys ...string) error {
	for _, key := range keys {
		if err := rcc.client.Del(context.Background(), key).Err(); err != nil {
			lv_log.Error(fmt.Sprintf("Del error: %v", err))
			return err
		}
	}
	return nil
}

func (rcc *RedisClient) HSet(key string, values ...interface{}) error {
	if err := rcc.client.HSet(context.Background(), key, values...).Err(); err != nil {
		lv_log.Error(fmt.Sprintf("HSet error: %v", err))
		return err
	}
	return nil
}

func (rcc *RedisClient) HGet(key, field string) (string, error) {
	data, err := rcc.client.HGet(context.Background(), key, field).Result()
	if err != nil {
		lv_log.Error(fmt.Sprintf("HGet error: %v", err))
		return "", err
	}
	return data, nil
}

func (rcc *RedisClient) HDel(key string, fields ...string) error {
	return rcc.client.HDel(context.Background(), key, fields...).Err()
}

func (rcc *RedisClient) HGetAll(key string) (map[string]string, error) {
	return rcc.client.HGetAll(context.Background(), key).Result()
}

func (rcc *RedisClient) Execute(script string, keys []string, args ...interface{}) (interface{}, error) {
	return rcc.client.Eval(context.TODO(), script, keys, args).Result()
}

func (rcc *RedisClient) Close() error {
	return rcc.client.Close()
}

func (rcc *RedisClient) Scan(cursor uint64, match string, count int64) ([]string, uint64, error) {
	return rcc.client.Scan(context.TODO(), cursor, match, count).Result()
}

func (rcc *RedisClient) GetRedis() *redis.Client {
	return rcc.client
}

func (rcc *RedisClient) CountKeysByPattern(pattern string) (int64, error) {
	var cursor uint64 = 0
	var count int64 = 0

	for {
		var keys []string
		var err error

		// 调用 SCAN 命令
		keys, cursor, err = rcc.Scan(cursor, pattern, 0)
		if err != nil {
			lv_log.Error(fmt.Sprintf("Scan error: %v", err))
			return 0, err
		}

		// 更新计数
		count += int64(len(keys))

		// 如果游标为 0，表示遍历完成
		if cursor == 0 {
			break
		}
	}

	return count, nil
}

func (rcc *RedisClient) GetKeysPage(pattern string, page int, pageSize int) (keys []string, total int, err error) {
	// 先获取总数
	total64, err := rcc.CountKeysByPattern(pattern)
	if err != nil {
		return nil, 0, err
	}
	total = int(total64)

	if total == 0 {
		return []string{}, 0, nil
	}

	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	if start >= total {
		return []string{}, total, nil
	}

	end := start + pageSize
	if end > total {
		end = total
	}
	//1. 去掉预分配：改为 var result []string，slice 长度完全由实际数据决定
	//2. SCAN count 动态调整：pageSize * 2，最少 100。这样：
	//- 小页码（如 pageSize=10）时用 100，减少往返次数
	//- 大页码（如 pageSize=500）时用 1000，提高效率
	//SCAN 的 count 参数只是"建议"值，Redis 可能返回更多或更少的 key。
	//3. 注意：Redis SCAN 返回的 key 是无序的，需要对结果排序的话可以在外部使用 sort.Strings()
	// 根据 pageSize 计算 SCAN 的 count 参数
	scanCount := int64(pageSize * 2)
	if scanCount < 100 {
		scanCount = 100
	}

	// 只扫描当前页需要的数据
	var result []string
	var count int
	var cursor uint64 = 0

	for {
		var batch []string
		batch, cursor, err = rcc.Scan(cursor, pattern, scanCount)
		if err != nil {
			lv_log.Error(fmt.Sprintf("Scan error: %v", err))
			return nil, 0, err
		}

		for _, key := range batch {
			if count >= start {
				result = append(result, key)
			}
			count++
			if count >= end {
				return result, total, nil
			}
		}

		if cursor == 0 {
			break
		}
	}

	return result, total, nil
}
