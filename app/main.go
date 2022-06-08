package main

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"

	config "github.com/m00lecule/stateful-scaling/config"
	models "github.com/m00lecule/stateful-scaling/models"
)

var router *gin.Engine

func main() {
	config.InitMetadata()
	config.InitDB()
	config.InitRedis()
	models.MigrateModels()

	tp, _ := config.TracerProvider(config.GetServiceName())

	otel.SetTracerProvider(tp)

	router = gin.Default()
	router.Use(otelgin.Middleware("stateful-app"))

	initializeRoutes()
	runServer(router)
}
