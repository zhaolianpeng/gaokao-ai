package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"gaokao-ai/backend/logging"
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
	logging.LogEvent("cache_connect", map[string]any{"addr": addr, "db": db, "ttlSeconds": int(ttl.Seconds())})
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
		unmarshalErr := json.Unmarshal(raw, &cached)
		if unmarshalErr == nil {
			logging.LogEvent("cache_get", map[string]any{"key": fullKey, "hit": true, "valuePreview": logging.PreviewString(string(raw), 512), "valueBytes": len(raw)})
			return cached, nil
		}
		logging.LogEvent("cache_get", map[string]any{"key": fullKey, "hit": true, "unmarshalError": unmarshalErr.Error(), "valuePreview": logging.PreviewString(string(raw), 512), "valueBytes": len(raw)})
	} else if err != redis.Nil {
		logging.LogEvent("cache_get", map[string]any{"key": fullKey, "hit": false, "error": err.Error()})
		return fetch()
	}
	logging.LogEvent("cache_get", map[string]any{"key": fullKey, "hit": false})

	value, err := fetch()
	if err != nil {
		logging.LogEvent("cache_fetch", map[string]any{"key": fullKey, "error": err.Error()})
		return zero, err
	}
	if payload, marshalErr := json.Marshal(value); marshalErr == nil {
		if setErr := cache.client.Set(ctx, fullKey, payload, cache.ttl).Err(); setErr != nil {
			logging.LogEvent("cache_set", map[string]any{"key": fullKey, "ttlSeconds": int(cache.ttl.Seconds()), "valueBytes": len(payload), "valuePreview": logging.PreviewString(string(payload), 512), "error": setErr.Error()})
		} else {
			logging.LogEvent("cache_set", map[string]any{"key": fullKey, "ttlSeconds": int(cache.ttl.Seconds()), "valueBytes": len(payload), "valuePreview": logging.PreviewString(string(payload), 512)})
		}
	} else {
		logging.LogEvent("cache_set", map[string]any{"key": fullKey, "marshalError": marshalErr.Error()})
	}
	return value, nil
}
