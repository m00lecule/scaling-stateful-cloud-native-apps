package main

import (
	"github.com/gin-gonic/gin"

	Config "github.com/m00lecule/stateful-scaling/config"
)

var router *gin.Engine

func main() {

	Config.InitDB()
	Config.InitMetadata()

	router = gin.Default()

	initializeRoutes()

	router.Run()
}
