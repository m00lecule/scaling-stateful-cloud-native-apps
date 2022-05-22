package config

import (
	"fmt"
	"github.com/caarlos0/env/v6"
	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
	redsyncredis "github.com/go-redsync/redsync/v4/redis"
	"github.com/go-redsync/redsync/v4/redis/goredis/v8"
	Log "github.com/sirupsen/logrus"
)

var RDB *redis.Client
var Redis *RedisConfig

var Pool redsyncredis.Pool
var Rs *redsync.Redsync

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
		Log.Warn(err)
	}
	return &c
}

func InitRedis() {
	Redis = getRedisConfig()

	RDB = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", Redis.Host, Redis.Port),
		Password: Redis.Password,
		DB:       Redis.DB,
	})

	Pool = goredis.NewPool(RDB)
	Rs = redsync.New(Pool)
}