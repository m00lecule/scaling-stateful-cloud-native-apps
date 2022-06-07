package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"

	config "github.com/m00lecule/stateful-scaling/config"
	models "github.com/m00lecule/stateful-scaling/models"
	log "github.com/sirupsen/logrus"
)

var tracer = otel.Tracer("gin-server")

func CreateCart(c *gin.Context) {
	var cart models.Cart = models.Cart{OwnedBy: config.Meta.Hostname}

	ctx := c.Request.Context()

	log.Info("Will create new cart")

	ctx, postgresSpan := tracer.Start(ctx, "postgres-create-cart")

	if dbc := config.DB.Create(&cart); dbc.Error != nil {
		c.AbortWithError(500, dbc.Error)
		return
	}

	postgresSpan.End()

	idStr := strconv.FormatUint(uint64(cart.ID), 10)

	config.InitMux(idStr)

	if err := config.InitRedisCart(ctx, idStr, tracer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"Error": err, "metadata": config.Meta})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payload":  idStr,
		"metadata": config.Meta,
	})
}

func UpdateCart(c *gin.Context) {
	var cartUpdate models.CartUpdate

	ctx := c.Request.Context()

	if err := c.BindJSON(&cartUpdate); err != nil {
		log.Error("Could not unmarshall data")
		c.JSON(http.StatusInternalServerError, gin.H{"Error": err, "metadata": config.Meta})
		return
	}

	id := c.Param("id")

	mx := config.GetMux(id)

	mx.Lock()
	defer mx.Unlock()

	if err := models.UpdateRedisCart(ctx, id, cartUpdate, tracer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"Error": err, "metadata": config.Meta})
		return
	}

	if err := models.UpdatePostgresCart(ctx, id, cartUpdate, tracer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"Error": err, "metadata": config.Meta})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"metadata": config.Meta,
	})
}

func GetCart(c *gin.Context) {
	var cartDetails = make(map[string]models.ProductDetails)

	ctx := c.Request.Context()
	idStr := c.Param("id")

	cartDetails, err := models.GetRedisCartDetails(ctx, idStr, tracer)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"Error": err,
			"metadata": config.Meta,
		})
		return
	}

	if cartDetails == nil {
		cartDetails, err = models.GetPostgresCartDetails(ctx, idStr, tracer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"Error": err,
				"metadata": config.Meta,
			})
			return
		}
	}

	u64, err := strconv.ParseUint(idStr, 10, 32)

	if err != nil {
		panic(err)
	}

	id := uint(u64)

	cart := models.Cart{ID: id, Content: cartDetails}

	c.JSON(http.StatusOK, gin.H{
		"payload":  cart,
		"metadata": config.Meta,
	})
}

func SubmitCart(c *gin.Context) {
	var cartDetails = make(map[string]models.ProductDetails)
	var p models.Product

	ctx := c.Request.Context()

	id := c.Param("id")

	mx := config.GetMux(id)

	mx.Lock()

	cartDetails, err := models.GetRedisCartDetails(ctx, id, tracer)

	if err != nil {
		mx.Unlock()
		c.JSON(http.StatusInternalServerError, gin.H{"Error": err})
		return
	}

	if cartDetails == nil {
		cartDetails, err = models.GetPostgresCartDetails(ctx, id, tracer)
		if err != nil {
			mx.Unlock()
			c.JSON(http.StatusInternalServerError, gin.H{"Error": err})
			return
		}
	}

	ctx, postgresSpan := tracer.Start(ctx, "postgres")

	tx := config.DB.Begin()

	for k, v := range cartDetails {
		p = models.Product{}
		if err = tx.Where("id = ?", k).First(&p).Error; err != nil {
			tx.Rollback()
			mx.Unlock()
			c.JSON(400, gin.H{"Error": err,
				"metadata": config.Meta,
			})
			return
		}

		if p.Stock < v.Count {
			msg := fmt.Sprintf("Product %s [%d] Stock [%d] is lower than request [%d]", p.Name, p.ID, p.Stock, v.Count)
			log.Debug(msg)
			tx.Rollback()
			mx.Unlock()
			c.JSON(444, gin.H{"Error": err,
				"metadata": config.Meta,
			})
			return
		}
		p.Stock -= v.Count

		if err = tx.Save(p).Error; err != nil {
			log.Error(err)
			tx.Rollback()
			mx.Unlock()
			c.JSON(400, gin.H{"Error": err,
				"metadata": config.Meta,
			})
			return
		}

		if err := tx.Model(&models.Cart{}).Where("id = ?", id).Updates(models.Cart{IsSubmitted: true}).Error; err != nil {
			log.Error(err)
			tx.Rollback()
			mx.Unlock()
			c.JSON(http.StatusInternalServerError, gin.H{"Error": err,
				"metadata": config.Meta,
			})
			return
		}
	}

	tx.Commit()

	postgresSpan.End()

	if config.Meta.IsStateful {
		_, err = config.RDB.Do(c.Request.Context(), "DEL", id).Result()

		if err != nil {
			mx.Unlock()
			c.JSON(http.StatusInternalServerError, gin.H{"Error": err,
				"metadata": config.Meta,
			})
			return
		}
	}

	mx.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"metadata": config.Meta,
	})
}
