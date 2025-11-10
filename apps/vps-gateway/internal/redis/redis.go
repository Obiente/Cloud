package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"vps-gateway/internal/logger"
)

type Client struct {
	client *redis.Client
}

func NewClient() (*Client, error) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		// Default to localhost if not set
		redisURL = "redis://localhost:6379"
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("[Redis] Connected to Redis at %s", redisURL)

	return &Client{
		client: client,
	}, nil
}

func (c *Client) Get(ctx context.Context, key string) (string, error) {
	if c.client == nil {
		return "", redis.Nil
	}
	return c.client.Get(ctx, key).Result()
}

func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if c.client == nil {
		return nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, expiration).Err()
}

func (c *Client) Keys(ctx context.Context, pattern string) ([]string, error) {
	if c.client == nil {
		return nil, fmt.Errorf("Redis client not initialized")
	}
	return c.client.Keys(ctx, pattern).Result()
}

func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

