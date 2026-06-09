package store

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type IdempotencyGate struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisClient(addr string, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db, DialTimeout: 3 * time.Second, ReadTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second})
}

func NewIdempotencyGate(client *redis.Client, ttl time.Duration) *IdempotencyGate {
	return &IdempotencyGate{client: client, ttl: ttl}
}

func (g *IdempotencyGate) FirstSeen(ctx context.Context, key string) (bool, error) {
	gateCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return g.client.SetNX(gateCtx, key, "1", g.ttl).Result()
}
