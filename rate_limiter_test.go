package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/suite"
)

type RateLimiter interface {
	Allow(ctx context.Context, id string) (bool, error)
}

type RateLimiterTestSuite struct {
	suite.Suite
	ctx         context.Context
	redisClient *redis.Client
	redisServer *miniredis.Miniredis
}

func (suite *RateLimiterTestSuite) SetupSuite() {
	mr, err := miniredis.Run()
	suite.Require().NoError(err)

	suite.redisServer = mr
	suite.redisClient = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	suite.ctx = context.Background()
}

func (suite *RateLimiterTestSuite) TearDownSuite() {
	suite.redisServer.Close()
	suite.redisClient.Close()
}

func (suite *RateLimiterTestSuite) TestTokenBucket() {
	limiter := &TokenBucket{
		redisClient: suite.redisClient,
		bucketSize:  5,
		refillRate:  1,
	}

	id := "random"

	suite.Run("Allows Within Limit", func() {
		for i := 0; i < 5; i++ {
			allowed, err := limiter.Allow(suite.ctx, id)
			suite.Require().NoError(err)
			suite.True(allowed)
		}
	})

	suite.Run("Denies Over Limit", func() {
		allowed, err := limiter.Allow(suite.ctx, id)
		suite.Require().NoError(err)
		suite.False(allowed)
	})

	suite.Run("Allows After Refill", func() {
		// Wait for bucket to be refilled (refillRate is 1)
		time.Sleep(1 * time.Second)

		allowed, err := limiter.Allow(suite.ctx, id)
		suite.Require().NoError(err)
		suite.True(allowed)

		// Next request should be denied
		allowed, err = limiter.Allow(suite.ctx, id)
		suite.Require().NoError(err)
		suite.False(allowed)
	})
}

func TestRateLimiterSuite(t *testing.T) {
	suite.Run(t, new(RateLimiterTestSuite))
}
