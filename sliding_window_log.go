package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type SlidingWindowLog struct {
	redisClient *redis.Client
	windowSize  int64
	limit       int64
}

func (sl *SlidingWindowLog) Allow(ctx context.Context, id string) (bool, error) {
	currentTime := time.Now().Unix()
	windowStart := currentTime - sl.windowSize
	windowStartStr := strconv.FormatInt(windowStart, 10)
	key := "rate_limit:" + id

	_, err := sl.redisClient.ZRemRangeByScore(ctx, key, "-inf", windowStartStr).Result()
	if err != nil {
		return false, err
	}

	count := sl.redisClient.ZCard(ctx, key).Val()
	if count >= sl.limit {
		return false, err
	}

	err = sl.redisClient.ZAdd(ctx, key, redis.Z{
		Score:  float64(currentTime),
		Member: fmt.Sprintf("%s:%d", id, currentTime),
	}).Err()
	if err != nil {
		return false, err
	}

	return true, nil
}
