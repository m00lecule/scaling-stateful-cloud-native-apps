package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	config "github.com/m00lecule/stateful-scaling/config"
)

func GetHealth(c *gin.Context) {
	if config.Meta.IsStateful {
		ctx := c.Request.Context()
		_, err := config.RDB.Ping(ctx).Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"Error": err,
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "UP",
	})
}
