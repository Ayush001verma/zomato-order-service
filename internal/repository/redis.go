package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient builds a Redis client and checks that Redis answers.
// It returns the client even if the check fails, so the caller can decide what to do.
func NewRedisClient(ctx context.Context, addr string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:        addr,
		DialTimeout: 2 * time.Second,
		ReadTimeout: 500 * time.Millisecond,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return client, client.Ping(pingCtx).Err()
}
