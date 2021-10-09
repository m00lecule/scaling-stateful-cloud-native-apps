package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/m00lecule/stateful-scaling/models"
)

var notesList = []models.Note{
	models.Note{ID: 1, Content: "Note 1 body"},
	models.Note{ID: 2, Content: "Note 2 body"},
}

func GetNote(c *gin.Context) {
	if noteID, err := strconv.Atoi(c.Param("note_id")); err == nil {

		if note, err := getNoteByID(noteID); err == nil {

			c.JSON(http.StatusOK, gin.H{
				"payload": note,
			})

		} else {
			c.AbortWithError(http.StatusNotFound, err)
		}

	} else {
		c.AbortWithStatus(http.StatusNotFound)
	}
}

func getNoteByID(id int) (*models.Note, error) {
	for _, a := range notesList {
		if a.ID == id {
			return &a, nil
		}
	}
	return nil, errors.New("Note not found")
}
