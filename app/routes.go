package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"

	handlers "github.com/m00lecule/stateful-scaling/handlers"
	models "github.com/m00lecule/stateful-scaling/models"
	log "github.com/sirupsen/logrus"
)

func initializeRoutes() {
	router.GET("/health", handlers.GetHealth)

	cartsRoutes := router.Group("/carts")
	{
		cartsRoutes.POST("/", handlers.CreateCart)
		cartsRoutes.PATCH("/:id", handlers.UpdateCart)
		cartsRoutes.GET("/:id", handlers.GetCart)
		cartsRoutes.POST("/:id/submit", handlers.SubmitCart)
	}
	productsRoutes := router.Group("/products")
	{
		productsRoutes.POST("/", handlers.CreateProduct)
		productsRoutes.GET("/:id", handlers.GetProduct)
		productsRoutes.DELETE("/:id", handlers.DeleteProduct)
	}
}

func runServer(router *gin.Engine) {
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	serverGracefulShutdown(srv)
}

func serverGracefulShutdown(srv *http.Server) {
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Info("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}

	models.OffloadModels(ctx)
	log.Info("Server shutdown finished")
}
