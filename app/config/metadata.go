package config

import (
	"github.com/caarlos0/env/v6"
	Log "github.com/sirupsen/logrus"
	"math/rand"
	"sync"
)

var Meta *Metadata
var MockedData string
var CartMux = map[string]*sync.Mutex{}

type Metadata struct {
	Hostname      string `env:"HOSTNAME" envDefault:"stateful-app"`
	LogLevel      string `env:"LOGLVL" envDefault:"info"`
	DataBytes     int    `env:"DATABYTES" envDefault:"512"`
	SessionMuxKey string `env:"GLOBAL_MUXNAME" envDefault:"sessions"`
}

func getMetadata() *Metadata {
	m := Metadata{}
	if err := env.Parse(&m); err != nil {
		Log.Warn(err)
	}
	return &m
}

func randomString(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func InitMetadata() {
	Meta = getMetadata()
	level, err := Log.ParseLevel(Meta.LogLevel)

	if err != nil {
		Log.SetLevel(Log.InfoLevel)
		Log.Info("Will set default loglevel - INFO")
	} else {
		Log.SetLevel(level)
	}
	MockedData = randomString(Meta.DataBytes)
}
