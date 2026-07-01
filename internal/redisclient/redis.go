package redisclient

import (
	"context"

	"github.com/redis/go-redis/v9"
)

const ScanStreamKey = "freshtrack:scan_events"

func NewClient(addr string) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: addr,
	})
}

func Ping(ctx context.Context, c *redis.Client) error {
	return c.Ping(ctx).Err()
}
