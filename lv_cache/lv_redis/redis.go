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