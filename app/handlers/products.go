package handlers

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"

	Config "github.com/m00lecule/stateful-scaling/config"
	Models "github.com/m00lecule/stateful-scaling/models"
)

func CreateProduct(c *gin.Context) {
	var p Models.Product

	err := c.BindJSON(&p)
	if err != nil {
		c.AbortWithError(400, err)
	}

	if dbc := Config.DB.Create(&p); dbc.Error != nil {
		c.AbortWithError(500, dbc.Error)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payload":  p,
		"metadata": Config.Meta,
	})
}

func GetProduct(c *gin.Context) {
	if productID, err := strconv.Atoi(c.Param("id")); err == nil {
		if product, err := getProductByID(productID); err == nil {

			c.JSON(http.StatusOK, gin.H{
				"payload":  product,
				"metadata": Config.Meta,
			})

		} else {
			c.AbortWithError(http.StatusNotFound, err)
		}

	} else {
		c.AbortWithStatus(http.StatusNotFound)
	}
}

func getProductByID(id int) (*Models.Product, error) {
	var p Models.Product
	if err := Config.DB.Where("id = ?", id).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}
