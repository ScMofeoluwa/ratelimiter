package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type SlidingWindowCounter struct {
	redisClient *redis.Client
	windowSize  int64
	bucketSize  int64
	limit       int64
}

func (sc *SlidingWindowCounter) Allow(ctx context.Context, id string) (bool, error) {
	currentTime := time.Now().Unix()
	key := "rate_limit:" + id

	state, err := sc.redisClient.HGetAll(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to get state from redis: %v\n", err)
	}

	var totalCount int64
	for _, val := range state {
		count, _ := strconv.ParseInt(val, 10, 64)
		totalCount += count
	}

	allowed := totalCount < sc.limit
	if allowed {
		currentBucket := currentTime / sc.bucketSize
		bucketField := strconv.FormatInt(currentBucket, 10)
		err := sc.redisClient.HIncrBy(ctx, key, bucketField, 1).Err()
		if err != nil {
			return false, err
		}

		err = sc.redisClient.HExpire(ctx, key, time.Second*time.Duration(sc.windowSize), bucketField).Err()
		if err != nil {
			return false, err
		}
	}

	return allowed, nil
}
