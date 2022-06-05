package models

import (
	"github.com/m00lecule/stateful-scaling/config"
)

func MigrateModels() {
	var DB = config.DB
	DB.AutoMigrate(&Product{}, &Cart{})
	DB.Migrator().CreateConstraint(&Product{}, "possitive_stock")
}
