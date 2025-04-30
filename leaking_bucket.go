package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type LeakingBucket struct {
	redisClient *redis.Client
	bucketSize  int64
	outflowRate float64
}

func (l *LeakingBucket) Allow(ctx context.Context, id string) (bool, error) {
	currentTime := time.Now().Unix()
	key := "rate_limit:" + id

	var queueSize int64
	var lastRefillTime int64

	state, err := l.redisClient.HGetAll(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to get state from redis: %v\n", err)
	}

	if len(state) == 0 {
		queueSize = 0
		lastRefillTime = currentTime
	} else {
		queueSize, _ = strconv.ParseInt(state["queueSize"], 10, 64)
		lastRefillTime, _ = strconv.ParseInt(state["lastRefillTime"], 10, 64)
	}

	timeElapsed := float64(currentTime - lastRefillTime)
	processedRequests := int64(timeElapsed * l.outflowRate)

	if processedRequests >= queueSize {
		queueSize = 0
	} else {
		queueSize -= processedRequests
	}

	allowed := false
	if queueSize < int64(l.bucketSize) {
		allowed = true
		queueSize++
	}

	err = l.redisClient.HSet(ctx, key, map[string]string{"queueSize": fmt.Sprintf("%d", queueSize), "lastRefillTime": strconv.FormatInt(currentTime, 10)}).Err()
	if err != nil {
		return false, fmt.Errorf("failed to save state to redis: %v\n", err)
	}

	return allowed, nil
}
