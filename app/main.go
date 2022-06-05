package main

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	Config "github.com/m00lecule/stateful-scaling/config"
	Models "github.com/m00lecule/stateful-scaling/models"
)

var router *gin.Engine

func main() {
	Config.InitMetadata()
	Config.InitRedis()
	Config.InitDB()
	Models.MigrateProducts()
	
	tp, _ := Config.TracerProvider()

	otel.SetTracerProvider(tp)
	
	router = gin.Default()
	router.Use(otelgin.Middleware("stateful-app"))
		
	initializeRoutes()
	router.Run()
}
