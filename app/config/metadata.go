package config

import (
	"fmt"
	"github.com/caarlos0/env/v6"
)

var Meta *Metadata

type Metadata struct {
	Hostname string `env:"HOSTNAME" envDefault:"stateful-app"`
}

func getMetadata() *Metadata {
	m := Metadata{}
	if err := env.Parse(&m); err != nil {
		fmt.Printf("%+v\n", err)
	}
	return &m
}

func InitMetadata() {
	Meta = getMetadata()
}
