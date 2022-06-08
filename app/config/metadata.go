package config

import (
	"math/rand"
	"sync"

	"github.com/caarlos0/env/v6"

	log "github.com/sirupsen/logrus"
)

var (
	Meta         *Metadata
	MockedData   string
	cartMuxMutex = sync.RWMutex{}
	CartMux      = map[string]*sync.Mutex{}
)

type Metadata struct {
	Hostname      string `env:"HOSTNAME" envDefault:"stateful-app"`
	LogLevel      string `env:"LOGLVL" envDefault:"info"`
	DataBytes     int    `env:"DATABYTES" envDefault:"512"`
	SessionMuxKey string `env:"GLOBAL_MUXNAME" envDefault:"sessions"`
	IsStateful    bool   `env:"IS_STATEFUL" envDefault:"true"`
}

func GetServiceName() string {
	if Meta.IsStateful {
		return "stateful-app"
	}
	return "stateless-app"
}

func getMetadata() *Metadata {
	m := Metadata{}
	if err := env.Parse(&m); err != nil {
		log.Warn(err)
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
	level, err := log.ParseLevel(Meta.LogLevel)

	if err != nil {
		log.SetLevel(log.InfoLevel)
		log.Info("Will set default loglevel - INFO")
	} else {
		log.SetLevel(level)
	}
	MockedData = randomString(Meta.DataBytes)
}

func InitMux(id string) {
	cartMuxMutex.Lock()
	defer cartMuxMutex.Unlock()
	CartMux[id] = &sync.Mutex{}
}

func GetMux(id string) *sync.Mutex {
	cartMuxMutex.RLock()
	defer cartMuxMutex.RUnlock()
	return CartMux[id]
}

func DelMux(id string) {
	cartMuxMutex.Lock()
	defer cartMuxMutex.Unlock()
	delete(CartMux, id)
}
