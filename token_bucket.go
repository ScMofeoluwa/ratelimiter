package ratelimit

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type TokenBucket struct {
	redisClient *redis.Client
	bucketSize  float64
	refillRate  float64
}

func (t *TokenBucket) Allow(ctx context.Context, id string) (bool, error) {
	currentTime := time.Now().Unix()
	key := "rate_limit:" + id

	var tokenCount float64
	var lastRefillTime int64

	state, err := t.redisClient.HGetAll(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to get state from redis: %v\n", err)
	}

	if len(state) == 0 {
		tokenCount = t.bucketSize
		lastRefillTime = currentTime
	} else {
		tokenCount, _ = strconv.ParseFloat(state["tokenCount"], 64)
		lastRefillTime, _ = strconv.ParseInt(state["lastRefillTime"], 10, 64)
	}

	timeElapsed := float64(currentTime - lastRefillTime)
	tokensToAdd := timeElapsed * t.refillRate

	allowed := false
	tokenCount = math.Min(t.bucketSize, tokenCount+float64(tokensToAdd))

	if tokenCount >= 1.0 {
		allowed = true
		tokenCount--
	}

	err = t.redisClient.HSet(ctx, key, map[string]string{"tokenCount": fmt.Sprintf("%.4f", tokenCount), "lastRefillTime": strconv.FormatInt(currentTime, 10)}).Err()
	if err != nil {
		return false, fmt.Errorf("failed to save state to redis: %v\n", err)
	}

	return allowed, nil
}
