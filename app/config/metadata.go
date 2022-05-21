package config

import (
	"github.com/caarlos0/env/v6"
	Log "github.com/sirupsen/logrus"
)

var Meta *Metadata

type Metadata struct {
	Hostname  string `env:"HOSTNAME" envDefault:"stateful-app"`
	LogLevel  string `env:"LOGLVL" envDefault:"debug"`
	DataBytes int    `env:"DATABYTES" envDefault:"2560000"`
}

func getMetadata() *Metadata {
	m := Metadata{}
	if err := env.Parse(&m); err != nil {
		Log.Warn("%+v\n", err)
	}
	return &m
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
}
