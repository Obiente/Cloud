package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"vps-gateway/internal/logger"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	client *redis.Client
}

func NewClient() (*Client, error) {
	redisURL := os.Getenv("REDIS_URL")

	// If REDIS_URL is not set, construct it from REDIS_HOST, REDIS_PORT, and REDIS_PASSWORD
	if redisURL == "" {
		host := os.Getenv("REDIS_HOST")
		if host == "" {
			host = "redis"
		}

		port := os.Getenv("REDIS_PORT")
		if port == "" {
			port = "6379"
		}

		password := os.Getenv("REDIS_PASSWORD")

		// Construct Redis URL with password if set
		// Format: redis://:password@host:port or redis://host:port
		if password != "" {
			redisURL = fmt.Sprintf("redis://:%s@%s:%s", password, host, port)
		} else {
			redisURL = fmt.Sprintf("redis://%s:%s", host, port)
		}
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
