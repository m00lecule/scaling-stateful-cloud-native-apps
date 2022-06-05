package models

import (
	"context"

	"github.com/m00lecule/stateful-scaling/config"

	log "github.com/sirupsen/logrus"
)

func MigrateModels() {
	var DB = config.DB
	DB.AutoMigrate(&Product{}, &Cart{})
	DB.Migrator().CreateConstraint(&Product{}, "possitive_stock")

	initCarts()
}

func OffloadModels(ctx context.Context) {
	log.Info("Starting offloading sessions")
	if err := offloadCarts(ctx); err != nil {
		panic(err)
	}
	log.Info("Offloading finished")
}
