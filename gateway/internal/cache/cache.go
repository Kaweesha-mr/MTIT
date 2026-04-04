package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// Entry represents a cached HTTP response.
type Entry struct {
	Status int               `json:"status"`
	Header map[string]string `json:"header"`
	Body   []byte            `json:"body"`
}

// Store wraps a Redis client for caching.
type Store struct {
	client  *redis.Client
	ttl     time.Duration
	enabled bool
}

// NewStore creates a Redis-backed cache store.
func NewStore(addr, password string, ttl time.Duration, enabled bool) *Store {
	if !enabled {
		return &Store{enabled: false}
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	return &Store{
		client:  client,
		ttl:     ttl,
		enabled: true,
	}
}

// Get retrieves a cached entry; hit is false when not found.
func (s *Store) Get(ctx context.Context, key string) (*Entry, bool, error) {
	if !s.enabled || s.client == nil {
		return nil, false, nil
	}

	raw, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var entry Entry
	if err := json.Unmarshal(raw, &entry); err != nil {
		return nil, false, err
	}

	return &entry, true, nil
}

// Set stores an entry with the configured TTL.
func (s *Store) Set(ctx context.Context, key string, entry *Entry) error {
	if !s.enabled || s.client == nil {
		return nil
	}

	payload, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	return s.client.Set(ctx, key, payload, s.ttl).Err()
}

// Delete removes a specific cache entry.
func (s *Store) Delete(ctx context.Context, key string) error {
	if !s.enabled || s.client == nil {
		return nil
	}

	return s.client.Del(ctx, key).Err()
}

// InvalidateByPrefix deletes all cache entries matching a prefix pattern.
// For example: "GET:/shelters:" will invalidate all GET requests cached for /shelters endpoints.
func (s *Store) InvalidateByPrefix(ctx context.Context, prefix string) error {
	if !s.enabled || s.client == nil {
		return nil
	}

	// Use SCAN to find keys matching the pattern and delete them
	iter := s.client.Scan(ctx, 0, prefix+"*", 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		if err := s.client.Del(ctx, key).Err(); err != nil {
			return err
		}
	}

	return iter.Err()
}

// Close closes the Redis client.
func (s *Store) Close() error {
	if !s.enabled || s.client == nil {
		return nil
	}

	return s.client.Close()
}
