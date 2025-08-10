package service

import (
	"context"
	"time"

	"convertpdfgo/pkg/logger"
	"convertpdfgo/storage"
)

type RedisService interface {
	SetX(ctx context.Context, key string, value interface{}, duration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	DeleteRefreshToken(ctx context.Context, userID string) error
	BlacklistToken(ctx context.Context, token string) error // <-- Yangi method
	IsTokenBlacklisted(ctx context.Context, token string) (bool, error)
}

type redisService struct {
	redis storage.IRedisStorage
	log   logger.ILogger
}

func NewRedisService(redis storage.IRedisStorage, log logger.ILogger) RedisService {
	return &redisService{
		redis: redis,
		log:   log,
	}
}

func (s *redisService) SetX(ctx context.Context, key string, value interface{}, duration time.Duration) error {
	s.log.Info("RedisService.SetX called", logger.String("key", key))
	return s.redis.SetX(ctx, key, value, duration)
}

func (s *redisService) Get(ctx context.Context, key string) (string, error) {
	return s.redis.Get(ctx, key)
}

func (s *redisService) DeleteRefreshToken(ctx context.Context, userID string) error {
	key := "refresh_token:" + userID
	return s.redis.Delete(ctx, key)
}

func (s *redisService) BlacklistToken(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	key := "jwt_blacklist:" + token
	return s.redis.SetX(ctx, key, "1", 7*24*time.Hour) // 7 kun block
}

func (s *redisService) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	key := "jwt_blacklist:" + token
	val, err := s.redis.Get(ctx, key)
	if err != nil && err.Error() != "redis: nil" {
		return false, err
	}
	return val == "1", nil
}
