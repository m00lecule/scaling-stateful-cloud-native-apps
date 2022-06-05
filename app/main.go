package main

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	config "github.com/m00lecule/stateful-scaling/config"
	models "github.com/m00lecule/stateful-scaling/models"
)

var router *gin.Engine

func main() {
	config.InitMetadata()
	config.InitRedis()
	config.InitDB()
	models.MigrateModels()
	
	tp, _ := config.TracerProvider()

	otel.SetTracerProvider(tp)
	
	router = gin.Default()
	router.Use(otelgin.Middleware("stateful-app"))
		
	initializeRoutes()
	router.Run()
}
