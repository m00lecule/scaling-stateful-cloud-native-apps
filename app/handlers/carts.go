package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	Config "github.com/m00lecule/stateful-scaling/config"
	Models "github.com/m00lecule/stateful-scaling/models"
	"github.com/orian/counters/global"
	Log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"net/http"
	"strconv"
	"sync"
)

var tracer = otel.Tracer("gin-server")

func CreateCart(c *gin.Context) {
	ctx := c.Request.Context()

	Log.Info("Will create new cart")
	counter := global.GetCounter("app")
	counter.Increment()

	id := counter.Value()
	idStr := strconv.FormatInt(id, 10)

	Config.CartMuxMutex.Lock()

	Config.CartMux[idStr] = &sync.Mutex{}

	Config.CartMuxMutex.Unlock()

	ctx, redisSpan := tracer.Start(ctx, "redis")

	err := Config.RDB.SAdd(c.Request.Context(), Config.Meta.SessionMuxKey, idStr).Err()

	if err != nil {
		Log.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"Error": err, "metadata": Config.Meta})
		return
	}

	bytes, _ := json.Marshal(map[string]string{})

	err = Config.RDB.Set(c.Request.Context(), idStr, bytes, 0).Err()

	if err != nil {
		Log.Error("Could not unmarshall data")
		c.JSON(http.StatusInternalServerError, gin.H{"Error": err, "metadata": Config.Meta})
		return
	}

	err = Config.RDB.Do(c.Request.Context(), "EXPIRE", id, Config.Redis.TTL).Err()

	redisSpan.End()

	if err != nil {
		Log.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"Error": err, "metadata": Config.Meta})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payload":  idStr,
		"metadata": Config.Meta,
	})
}

func UpdateCart(c *gin.Context) {
	var cartUpdate Models.CartUpdate

	ctx := c.Request.Context()

	var cart = make(map[string]Models.ProductDetails)

	err := c.BindJSON(&cartUpdate)

	if err != nil {
		Log.Error("Could not unmarshall data")
		c.JSON(http.StatusInternalServerError, gin.H{"Error": err, "metadata": Config.Meta})
		return
	}

	Log.Info(cartUpdate.Details)

	id := c.Param("id")

	Config.CartMuxMutex.RLock()
	mx := Config.CartMux[id]
	Config.CartMuxMutex.RUnlock()

	mx.Lock()

	currentCartBytes, err := Config.RDB.Get(c.Request.Context(), id).Result()

	err = json.Unmarshal([]byte(currentCartBytes), &cart)

	for id, delta := range cartUpdate.Details {
		Log.Debug(id, " - ", delta)
		if value, ok := cart[id]; ok {
			value.Count = value.Count + delta

			data := []string{}

			for i := uint(0); i < value.Count; i++ {
				data = append(data, Config.MockedData)
			}

			value.Data = data
			cart[id] = value
		} else {
			data := []string{}

			for i := uint(0); i < delta; i++ {
				data = append(data, Config.MockedData)
			}

			cart[id] = Models.ProductDetails{Count: delta, Data: data}
		}
	}

	bytes, err := json.Marshal(cart)

	ctx, redisSpan := tracer.Start(ctx, "redis")

	err = Config.RDB.Set(c.Request.Context(), id, bytes, 0).Err()

	redisSpan.End()

	if err != nil {
		Log.Warn("Could not connect to redis")
		mx.Unlock()
		panic(err)
	}

	c.JSON(http.StatusOK, gin.H{
		"metadata": Config.Meta,
	})

	mx.Unlock()
}

func GetCart(c *gin.Context) {
	var cartDetails = make(map[string]Models.ProductDetails)

	ctx := c.Request.Context()

	id := c.Param("id")

	ctx, redisSpan := tracer.Start(ctx, "redis")

	cartDetailsBytes, err := Config.RDB.Get(c.Request.Context(), id).Result()

	redisSpan.End()

	err = json.Unmarshal([]byte(cartDetailsBytes), &cartDetails)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"Error": err,
			"metadata": Config.Meta,
		})
		return
	}

	cart := Models.Cart{ID: id, Content: cartDetails}

	c.JSON(http.StatusOK, gin.H{
		"payload":  cart,
		"metadata": Config.Meta,
	})
}

func SubmitCart(c *gin.Context) {
	var cartDetails = make(map[string]Models.ProductDetails)
	var p Models.Product

	ctx := c.Request.Context()

	id := c.Param("id")
	Config.CartMuxMutex.RLock()
	mx := Config.CartMux[id]
	Config.CartMuxMutex.RUnlock()

	mx.Lock()

	ctx, redisSpan := tracer.Start(ctx, "redis")

	cartDetailsBytes, err := Config.RDB.Get(c.Request.Context(), id).Result()

	redisSpan.End()

	err = json.Unmarshal([]byte(cartDetailsBytes), &cartDetails)

	if err != nil {
		mx.Unlock()
		c.JSON(http.StatusInternalServerError, gin.H{"Error": err})
		return
	}

	ctx, postgresSpan := tracer.Start(ctx, "postgres")

	tx := Config.DB.Begin()

	for k, v := range cartDetails {
		p = Models.Product{}
		if err = tx.Where("id = ?", k).First(&p).Error; err != nil {
			tx.Rollback()
			mx.Unlock()
			c.JSON(400, gin.H{"Error": err,
				"metadata": Config.Meta,
			})
			return
		}

		if p.Stock < v.Count {
			msg := fmt.Sprintf("Product %s [%d] Stock [%d] is lower than request [%d]", p.Name, p.ID, p.Stock, v.Count)
			Log.Debug(msg)
			tx.Rollback()
			mx.Unlock()
			c.JSON(444, gin.H{"Error": err,
				"metadata": Config.Meta,
			})
			return
		}
		p.Stock -= v.Count

		if err = tx.Save(p).Error; err != nil {
			Log.Error(err)
			tx.Rollback()
			mx.Unlock()
			c.JSON(400, gin.H{"Error": err,
				"metadata": Config.Meta,
			})
			return
		}
	}

	tx.Commit()

	postgresSpan.End()

	_, err = Config.RDB.Do(c.Request.Context(), "DEL", id).Result()

	if err != nil {
		mx.Unlock()
		c.JSON(http.StatusInternalServerError, gin.H{"Error": err,
			"metadata": Config.Meta,
		})
		return
	}

	mx.Unlock()

	Config.CartMuxMutex.Lock()
	delete(Config.CartMux, id)
	Config.CartMuxMutex.Unlock()

	err = Config.RDB.SRem(c.Request.Context(), Config.Meta.SessionMuxKey, id).Err()

	c.JSON(http.StatusOK, gin.H{
		"metadata": Config.Meta,
	})
}
