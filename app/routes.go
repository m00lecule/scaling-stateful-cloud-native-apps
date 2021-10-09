package main

import "github.com/m00lecule/stateful-scaling/handlers"

func initializeRoutes() {
	notesRoutes := router.Group("/notes")
	{
		notesRoutes.GET("/:note_id", handlers.GetNote)
	}
}
