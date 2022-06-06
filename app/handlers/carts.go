package handlers

import (
	"encoding/json"
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

	ctx, redisSpan := tracer.Start(ctx, "redis")

	bytes, _ := json.Marshal(map[string]string{})

	err := config.RDB.Set(ctx, idStr, bytes, 0).Err()

	if err != nil {
		log.Error("Could not unmarshall data")
		c.JSON(http.StatusInternalServerError, gin.H{"Error": err, "metadata": config.Meta})
		return
	}

	err = config.RDB.Do(ctx, "EXPIRE", idStr, config.Redis.TTL).Err()

	redisSpan.End()

	if err != nil {
		log.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"Error": err, "metadata": config.Meta})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payload":  idStr,
		"metadata": config.Meta,
	})
}

func UpdateCart(c *gin.Context) {
	var cart = make(map[string]models.ProductDetails)
	var cartUpdate models.CartUpdate

	ctx := c.Request.Context()

	err := c.BindJSON(&cartUpdate)

	if err != nil {
		log.Error("Could not unmarshall data")
		c.JSON(http.StatusInternalServerError, gin.H{"Error": err, "metadata": config.Meta})
		return
	}

	id := c.Param("id")

	mx := config.GetMux(id)

	mx.Lock()

	currentCartBytes, err := config.RDB.Get(c.Request.Context(), id).Result()

	err = json.Unmarshal([]byte(currentCartBytes), &cart)

	for id, delta := range cartUpdate.Details {
		log.Debug(id, " - ", delta)
		if value, ok := cart[id]; ok {
			value.Count = value.Count + delta

			data := []string{}

			for i := uint(0); i < value.Count; i++ {
				data = append(data, config.MockedData)
			}

			value.Data = data
			cart[id] = value
		} else {
			data := []string{}

			for i := uint(0); i < delta; i++ {
				data = append(data, config.MockedData)
			}

			cart[id] = models.ProductDetails{Count: delta, Data: data}
		}
	}

	bytes, err := json.Marshal(cart)

	ctx, redisSpan := tracer.Start(ctx, "redis")

	err = config.RDB.Set(c.Request.Context(), id, bytes, 0).Err()

	redisSpan.End()

	if err != nil {
		log.Warn("Could not connect to redis")
		mx.Unlock()
		panic(err)
	}

	c.JSON(http.StatusOK, gin.H{
		"metadata": config.Meta,
	})

	mx.Unlock()
}

func GetCart(c *gin.Context) {
	var cartDetails = make(map[string]models.ProductDetails)

	ctx := c.Request.Context()
	idStr := c.Param("id")
	ctx, redisSpan := tracer.Start(ctx, "redis")

	cartDetailsBytes, err := config.RDB.Get(c.Request.Context(), idStr).Result()

	redisSpan.End()

	err = json.Unmarshal([]byte(cartDetailsBytes), &cartDetails)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"Error": err,
			"metadata": config.Meta,
		})
		return
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

	ctx, redisSpan := tracer.Start(ctx, "redis")

	cartDetailsBytes, err := config.RDB.Get(c.Request.Context(), id).Result()

	redisSpan.End()

	err = json.Unmarshal([]byte(cartDetailsBytes), &cartDetails)

	if err != nil {
		mx.Unlock()
		c.JSON(http.StatusInternalServerError, gin.H{"Error": err})
		return
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

	_, err = config.RDB.Do(c.Request.Context(), "DEL", id).Result()

	if err != nil {
		mx.Unlock()
		c.JSON(http.StatusInternalServerError, gin.H{"Error": err,
			"metadata": config.Meta,
		})
		return
	}

	mx.Unlock()

	config.DelMux(id)

	c.JSON(http.StatusOK, gin.H{
		"metadata": config.Meta,
	})
}
