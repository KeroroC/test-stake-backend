package cache

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Get 从 Redis 读取并反序列化缓存值。命中返回 (value, true, nil)。
func Get[T any](ctx context.Context, rdb *redis.Client, key string) (T, bool, error) {
	var zero T
	data, err := rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return zero, false, nil
		}
		return zero, false, fmt.Errorf("cache get %s: %w", key, err)
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return zero, false, fmt.Errorf("cache unmarshal %s: %w", key, err)
	}

	return result, true, nil
}

// Set 序列化并存储值到 Redis，带 TTL。
func Set(ctx context.Context, rdb *redis.Client, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache marshal %s: %w", key, err)
	}

	if err := rdb.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("cache set %s: %w", key, err)
	}

	return nil
}

// DeleteByPrefix 通过 SCAN + DEL 删除匹配前缀的所有 key。
func DeleteByPrefix(ctx context.Context, rdb *redis.Client, prefix string) error {
	var cursor uint64
	for {
		keys, nextCursor, err := rdb.Scan(ctx, cursor, prefix+"*", 100).Result()
		if err != nil {
			return fmt.Errorf("cache scan %s: %w", prefix, err)
		}

		if len(keys) > 0 {
			if err := rdb.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("cache del %s: %w", prefix, err)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

// BuildListKey 根据事件类型和查询参数生成缓存 key。
// 格式："{eventType}:list:{md5(json)}"
func BuildListKey(eventType string, query any) string {
	data, err := json.Marshal(query)
	if err != nil {
		log.Printf("cache: failed to marshal query for key building: %v", err)
		return fmt.Sprintf("%s:list:error", eventType)
	}
	hash := md5.Sum(data)
	return fmt.Sprintf("%s:list:%x", eventType, hash)
}
