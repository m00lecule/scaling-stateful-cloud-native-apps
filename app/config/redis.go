package config

import (
	"context"
	"fmt"
	"encoding/json"

	"github.com/caarlos0/env/v6"
	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
	"go.opentelemetry.io/otel/trace"

	redsyncredis "github.com/go-redsync/redsync/v4/redis"
	log "github.com/sirupsen/logrus"
)

var (
	RDB   *redis.Client
	Redis *RedisConfig
	Pool  redsyncredis.Pool
	Rs    *redsync.Redsync
)

type RedisConfig struct {
	Host     string `env:"REDIS_HOST" envDefault:"localhost"`
	Port     int    `env:"REDIS_PORT" envDefault:"6379"`
	Password string `env:"REDIS_PASSWORD" envDefault:""`
	TTL      int    `env:"REDIS_KEYS_TTL" envDefault:"120"`
	DB       int    `env:"REDIS_DB" envDefault:"0"`
}

func getRedisConfig() *RedisConfig {
	c := RedisConfig{}
	if err := env.Parse(&c); err != nil {
		log.Warn(err)
	}
	return &c
}

func InitRedis() {
	if ! Meta.IsStateful {
		log.Info("Skipping Redis client setup")
		return
	}

	Redis = getRedisConfig()

	RDB = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", Redis.Host, Redis.Port),
		Password: Redis.Password,
		DB:       Redis.DB,
	})

	Pool = goredis.NewPool(RDB)
	Rs = redsync.New(Pool)
}

func InitRedisCart(ctx context.Context, id string, tracer trace.Tracer) error {
	if ! Meta.IsStateful {
		return nil
	}

	ctx, redisSpan := tracer.Start(ctx, "redis")
	
	bytes, _ := json.Marshal(map[string]string{})

	if err := RDB.Set(ctx, id, bytes, 0).Err(); err != nil {
		return err
	}

	if err := RDB.Do(ctx, "EXPIRE", id, Redis.TTL).Err(); err != nil {
		return err
	}

	redisSpan.End()

	return nil
}
