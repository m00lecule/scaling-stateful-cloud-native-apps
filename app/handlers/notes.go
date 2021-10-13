package handlers

import (
	"net/http"
	"strconv"
	"github.com/gin-gonic/gin"
	
	Config "github.com/m00lecule/stateful-scaling/config"
	Models "github.com/m00lecule/stateful-scaling/models"
)

func CreateNote(c *gin.Context) {
	var m  Models.Note
	
	err := c.BindJSON(&m)
	if err != nil {
		c.AbortWithError(400, err)
	}
	
	if dbc := Config.DB.Create(&m); dbc.Error != nil {
		c.AbortWithError(500, dbc.Error)
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"payload": m,
	})
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

func getNoteByID(id int) (*Models.Note, error) {
	var n Models.Note
	if err := Config.DB.Where("id = ?", id).First(&n).Error; err != nil {
		return nil, err
	}
	return &n, nil
}