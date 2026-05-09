package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type ResultCache struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

func NewResultCache(addr, password string, db int, ttl time.Duration) (*ResultCache, error) {
	if strings.TrimSpace(addr) == "" {
		return &ResultCache{}, nil
	}
	if ttl <= 0 {
		ttl = 6 * time.Hour
	}
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return &ResultCache{client: client, ttl: ttl, prefix: "gaokao:"}, nil
}

func (c *ResultCache) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}

func (c *ResultCache) enabled() bool {
	return c != nil && c.client != nil
}

func rememberJSON[T any](ctx context.Context, cache *ResultCache, key string, fetch func() (T, error)) (T, error) {
	var zero T
	if cache == nil || !cache.enabled() {
		return fetch()
	}

	fullKey := cache.prefix + key
	if raw, err := cache.client.Get(ctx, fullKey).Bytes(); err == nil {
		var cached T
		if unmarshalErr := json.Unmarshal(raw, &cached); unmarshalErr == nil {
			return cached, nil
		}
	} else if err != redis.Nil {
		return fetch()
	}

	value, err := fetch()
	if err != nil {
		return zero, err
	}
	if payload, marshalErr := json.Marshal(value); marshalErr == nil {
		_ = cache.client.Set(ctx, fullKey, payload, cache.ttl).Err()
	}
	return value, nil
}
