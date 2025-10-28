package database

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache() *RedisCache {
	return &RedisCache{}
}

func (r *RedisCache) Connect() error {
	opt, err := redis.ParseURL("redis://localhost:6379")
	if err != nil {
		return err
	}

	r.client = redis.NewClient(opt)

	// Test connection
	ctx := context.Background()
	_, err = r.client.Ping(ctx).Result()
	return err
}

func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	if r.client == nil {
		return "", redis.Nil
	}
	return r.client.Get(ctx, key).Result()
}

func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if r.client == nil {
		return nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, key, data, expiration).Err()
}

func (r *RedisCache) Delete(ctx context.Context, keys ...string) error {
	if r.client == nil {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}

func (r *RedisCache) Exists(ctx context.Context, key string) bool {
	if r.client == nil {
		return false
	}
	exists, _ := r.client.Exists(ctx, key).Result()
	return exists > 0
}
