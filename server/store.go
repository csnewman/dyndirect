package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Store interface {
	SetACMEChallengeTokens(ctx context.Context, id uuid.UUID, tokens []string) error

	GetACMEChallengeTokens(ctx context.Context, id uuid.UUID) ([]string, error)

	IncrementStat(ctx context.Context, key string, value int64)
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
	if errors.Is(err, redis.Nil) {
		return res, nil
	} else if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(val), &res); err != nil {
		return nil, err
	}

	return res, nil
}

func (s *RedisStore) IncrementStat(_ context.Context, _ string, _ int64) {
}

type memChallenge struct {
	stamp  time.Time
	values []string
}

type MemStore struct {
	mu         sync.Mutex
	challenges map[uuid.UUID]memChallenge
	logger     *zap.SugaredLogger
	stats      map[string]int64
}

func NewMemStore(logger *zap.SugaredLogger) (*MemStore, error) {
	stats := map[string]int64{}

	file, err := os.Open("cache/stats.json")
	if err == nil {
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(data, &stats); err != nil {
			return nil, err
		}
	} else {
		logger.Errorw("Failed to read stats file", "err", err)
	}

	return &MemStore{
		challenges: map[uuid.UUID]memChallenge{},
		logger:     logger,
		stats:      stats,
	}, nil
}

func (s *MemStore) SetACMEChallengeTokens(_ context.Context, id uuid.UUID, tokens []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.challenges[id] = memChallenge{
		stamp:  time.Now(),
		values: tokens,
	}

	return nil
}

func (s *MemStore) GetACMEChallengeTokens(_ context.Context, id uuid.UUID) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.challenges[id]
	if !ok {
		return nil, nil
	}

	return entry.values, nil
}

func (s *MemStore) IncrementStat(_ context.Context, key string, value int64) {
	go func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		old := s.stats[key]

		s.stats[key] = old + value
	}()
}

func (s *MemStore) AutoCleanup() {
	for range time.Tick(30 * time.Second) {
		s.Clean()
	}
}

func (s *MemStore) Clean() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	oldest := now.Add(-30 * time.Minute)

	type kv struct {
		key   uuid.UUID
		value time.Time
	}

	//nolint:prealloc
	var ss []kv

	removed := 0

	for k, v := range s.challenges {
		if v.stamp.Unix() < oldest.Unix() {
			delete(s.challenges, k)

			removed++

			continue
		}

		ss = append(ss, kv{k, v.stamp})
	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].value.Unix() > ss[j].value.Unix()
	})

	for i := len(s.challenges) - 1; i >= 1000000; i-- {
		delete(s.challenges, ss[i].key)

		removed++
	}

	s.logger.Debugw("Store cleaned", "acme_active", len(s.challenges), "acme_removed", removed, "stats", s.stats)

	encoded, err := json.MarshalIndent(s.stats, "", "    ")
	if err != nil {
		s.logger.Warnw("Failed to encode stats", "err", err)

		return
	}

	if err := os.WriteFile("cache/stats.json", encoded, 0o644); err != nil {
		s.logger.Warnw("Failed to write stats", "err", err)
	}
}
