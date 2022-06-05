package config

import (
	"fmt"

	"github.com/caarlos0/env/v6"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

type PostgresConfig struct {
	Host        string `env:"POSTGRES_HOST" envDefault:"localhost"`
	Port        int    `env:"POSTGRES_PORT" envDefault:"5432"`
	User        string `env:"POSTGRES_USER" envDefault:"gorm"`
	Password    string `env:"POSTGRES_PASSWORD" envDefault:"gorm"`
	Database    string `env:"POSTGRES_DB" envDefault:"gorm"`
	SSLMode     string `env:"POSTGRES_SSL_MODE" envDefault:"disable"`
	MaxCons     int    `env:"POSTGRES_MAX_CONS" envDefault:"20"`
	MaxIdleCons int    `env:"POSTGRES_MAX_IDLE_CONS" envDefault:"5"`
}

func getPostgresConfig() *PostgresConfig {
	c := PostgresConfig{}
	if err := env.Parse(&c); err != nil {
		fmt.Printf("%+v\n", err)
	}
	return &c
}

func (c *PostgresConfig) getDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode)
}

func InitDB() {
	c := getPostgresConfig()

	var err error
	dsn := c.getDSN()
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		panic(err)
	}

	conf, err := DB.DB()

	if err != nil {
		panic(err)
	}

	conf.SetMaxIdleConns(c.MaxIdleCons)
	conf.SetMaxOpenConns(c.MaxCons)

	if err != nil {
		panic(err)
	}
}
