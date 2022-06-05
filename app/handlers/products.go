package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	config "github.com/m00lecule/stateful-scaling/config"
	models "github.com/m00lecule/stateful-scaling/models"
)

func CreateProduct(c *gin.Context) {
	var p models.Product

	err := c.BindJSON(&p)
	if err != nil {
		c.AbortWithError(400, err)
	}

	if dbc := config.DB.Create(&p); dbc.Error != nil {
		c.AbortWithError(500, dbc.Error)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payload":  p,
		"metadata": config.Meta,
	})
}

func GetProduct(c *gin.Context) {
	if productID, err := strconv.Atoi(c.Param("id")); err == nil {
		if product, err := getProductByID(productID); err == nil {

			c.JSON(http.StatusOK, gin.H{
				"payload":  product,
				"metadata": config.Meta,
			})

		} else {
			c.AbortWithError(http.StatusNotFound, err)
		}
	} else {
		c.AbortWithStatus(http.StatusNotFound)
	}
}

func DeleteProduct(c *gin.Context) {
	if productID, err := strconv.Atoi(c.Param("id")); err == nil {
		if product, err := getProductByID(productID); err == nil {
			err = models.DelProduct(product)

			if err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
			}

			c.JSON(http.StatusOK, gin.H{
				"metadata": config.Meta,
			})

		} else {
			c.AbortWithError(http.StatusNotFound, err)
		}
	} else {
		c.AbortWithStatus(http.StatusNotFound)
	}
}

func getProductByID(id int) (*models.Product, error) {
	var p models.Product
	if err := config.DB.Where("id = ?", id).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}
