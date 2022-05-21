package main

import (
	"github.com/gin-gonic/gin"
	Config "github.com/m00lecule/stateful-scaling/config"
	Models "github.com/m00lecule/stateful-scaling/models"
)

var router *gin.Engine

func main() {
	Config.InitMetadata()
	Config.InitRedis()
	Config.InitDB()

	Models.MigrateProducts()

	router = gin.Default()
	initializeRoutes()
	router.Run()
}
