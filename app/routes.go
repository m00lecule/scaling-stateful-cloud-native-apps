package main

import Handlers "github.com/m00lecule/stateful-scaling/handlers"

func initializeRoutes() {
	router.GET("/health", Handlers.GetHealth)

	cartsRoutes := router.Group("/carts")
	{
		cartsRoutes.POST("/", Handlers.CreateCart)
		cartsRoutes.PATCH("/:id", Handlers.UpdateCart)
		cartsRoutes.GET("/:id", Handlers.GetCart)
		cartsRoutes.POST("/:id/submit", Handlers.SubmitCart)
	}
	productsRoutes := router.Group("/products")
	{
		productsRoutes.POST("/", Handlers.CreateProduct)
		productsRoutes.GET("/:id", Handlers.GetProduct)
		productsRoutes.DELETE("/:id", Handlers.DeleteProduct)
	}
}
