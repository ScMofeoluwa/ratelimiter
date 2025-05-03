package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type FixedWindow struct {
	redisClient *redis.Client
	windowSize  int64
	limit       int64
}

func (f *FixedWindow) Allow(ctx context.Context, id string) (bool, error) {
	currentTime := time.Now().Unix()
	currentWindow := currentTime / f.windowSize
	key := fmt.Sprintf("rate_limit:%s:%v", id, currentWindow)

	count := f.redisClient.Incr(ctx, key).Val()

	if count == 1 {
		err := f.redisClient.Expire(ctx, key, time.Second*time.Duration(f.windowSize)).Err()
		if err != nil {
			return false, err
		}
	}

	return count < f.limit, nil
}
