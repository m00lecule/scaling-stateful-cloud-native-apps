package main

import Handlers "github.com/m00lecule/stateful-scaling/handlers"

func initializeRoutes() {
	notesRoutes := router.Group("/notes")
	{
		notesRoutes.POST("/", Handlers.CreateNote)
		notesRoutes.GET("/:note_id", Handlers.GetNote)
	}
}
