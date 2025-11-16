package database

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
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
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return err
	}

	// Optimize connection pool settings for better performance
	// PoolSize: Number of connections per CPU core (default is 10 * numCPU)
	// MinIdleConns: Minimum idle connections to maintain
	if opt.PoolSize == 0 {
		opt.PoolSize = 20 // Increased from default for better concurrency
	}
	if opt.MinIdleConns == 0 {
		opt.MinIdleConns = 5 // Keep some connections warm
	}

	// Connection timeouts
	if opt.DialTimeout == 0 {
		opt.DialTimeout = 5 * time.Second
	}
	if opt.ReadTimeout == 0 {
		opt.ReadTimeout = 3 * time.Second
	}
	if opt.WriteTimeout == 0 {
		opt.WriteTimeout = 3 * time.Second
	}

	// Connection pool timeouts
	if opt.PoolTimeout == 0 {
		opt.PoolTimeout = 4 * time.Second
	}

	// Retry configuration
	if opt.MaxRetries == 0 {
		opt.MaxRetries = 3
	}
	if opt.MinRetryBackoff == 0 {
		opt.MinRetryBackoff = 8 * time.Millisecond
	}
	if opt.MaxRetryBackoff == 0 {
		opt.MaxRetryBackoff = 512 * time.Millisecond
	}

	// Allow configuration via environment variables
	if poolSizeStr := os.Getenv("REDIS_POOL_SIZE"); poolSizeStr != "" {
		if poolSize, err := strconv.Atoi(poolSizeStr); err == nil && poolSize > 0 {
			opt.PoolSize = poolSize
		}
	}
	if minIdleStr := os.Getenv("REDIS_MIN_IDLE_CONNS"); minIdleStr != "" {
		if minIdle, err := strconv.Atoi(minIdleStr); err == nil && minIdle >= 0 {
			opt.MinIdleConns = minIdle
		}
	}

	r.client = redis.NewClient(opt)

	// Test connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = r.client.Ping(ctx).Result()
	return err
}

func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	if r.client == nil {
		return "", redis.Nil
	}
	return r.client.Get(ctx, key).Result()
}

// GetWithTTL retrieves a value and its remaining TTL
func (r *RedisCache) GetWithTTL(ctx context.Context, key string) (string, time.Duration, error) {
	if r.client == nil {
		return "", 0, redis.Nil
	}
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return "", 0, err
	}
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return val, 0, err
	}
	return val, ttl, nil
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

// SetNX sets a key only if it doesn't exist (useful for distributed locks)
func (r *RedisCache) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	if r.client == nil {
		return false, nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return false, err
	}

	return r.client.SetNX(ctx, key, data, expiration).Result()
}

// MGet retrieves multiple keys at once (more efficient than multiple Gets)
func (r *RedisCache) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	if r.client == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}
	if len(keys) == 0 {
		return []interface{}{}, nil
	}
	return r.client.MGet(ctx, keys...).Result()
}

// MSet sets multiple key-value pairs at once (more efficient than multiple Sets)
func (r *RedisCache) MSet(ctx context.Context, pairs map[string]interface{}, expiration time.Duration) error {
	if r.client == nil {
		return nil
	}
	if len(pairs) == 0 {
		return nil
	}

	// Use pipeline for better performance
	pipe := r.client.Pipeline()
	for key, value := range pairs {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		pipe.Set(ctx, key, data, expiration)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisCache) Delete(ctx context.Context, keys ...string) error {
	if r.client == nil {
		return nil
	}
	if len(keys) == 0 {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}

// DeletePattern deletes all keys matching a pattern (use with caution)
func (r *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
	if r.client == nil {
		return nil
	}
	// Use SCAN instead of KEYS for better performance on large datasets
	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if len(keys) > 0 {
		return r.client.Del(ctx, keys...).Err()
	}
	return nil
}

func (r *RedisCache) Exists(ctx context.Context, key string) bool {
	if r.client == nil {
		return false
	}
	exists, _ := r.client.Exists(ctx, key).Result()
	return exists > 0
}

// Keys returns all keys matching a pattern
// WARNING: Use with caution on large datasets - prefer ScanPattern
func (r *RedisCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	if r.client == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}
	return r.client.Keys(ctx, pattern).Result()
}

// ScanPattern scans keys matching a pattern (safer for large datasets)
func (r *RedisCache) ScanPattern(ctx context.Context, pattern string, count int64) ([]string, error) {
	if r.client == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}
	if count <= 0 {
		count = 100 // Default scan count
	}

	var keys []string
	iter := r.client.Scan(ctx, 0, pattern, count).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

// Increment increments a key's value (useful for counters)
func (r *RedisCache) Increment(ctx context.Context, key string) (int64, error) {
	if r.client == nil {
		return 0, fmt.Errorf("redis client not initialized")
	}
	return r.client.Incr(ctx, key).Result()
}

// IncrementBy increments a key's value by a specific amount
func (r *RedisCache) IncrementBy(ctx context.Context, key string, value int64) (int64, error) {
	if r.client == nil {
		return 0, fmt.Errorf("redis client not initialized")
	}
	return r.client.IncrBy(ctx, key, value).Result()
}

// Expire sets expiration on an existing key
func (r *RedisCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if r.client == nil {
		return nil
	}
	return r.client.Expire(ctx, key, expiration).Err()
}

// Close closes the Redis connection
func (r *RedisCache) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}
