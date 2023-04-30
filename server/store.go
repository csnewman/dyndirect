package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"time"
)

type Store interface {
	SetACMEChallengeTokens(ctx context.Context, id uuid.UUID, tokens []string) error

	GetACMEChallengeTokens(ctx context.Context, id uuid.UUID) ([]string, error)
}

type RedisStore struct {
	rdb *redis.Client
}

func NewRedisStore(cfg Config) *RedisStore {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Username: cfg.RedisUser,
		Password: cfg.RedisPass,
		DB:       cfg.RedisDB,
	})

	return &RedisStore{
		rdb: rdb,
	}
}

func (s *RedisStore) SetACMEChallengeTokens(ctx context.Context, id uuid.UUID, tokens []string) error {
	val, err := json.Marshal(tokens)
	if err != nil {
		return err
	}

	return s.rdb.Set(ctx, fmt.Sprintf("%s-acme-challenge", id), string(val), time.Hour).Err()
}

func (s *RedisStore) GetACMEChallengeTokens(ctx context.Context, id uuid.UUID) ([]string, error) {
	var res []string

	val, err := s.rdb.Get(ctx, fmt.Sprintf("%s-acme-challenge", id)).Result()
	if err == redis.Nil {
		return res, nil
	} else if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(val), &res); err != nil {
		return nil, err
	}

	return res, nil
}
