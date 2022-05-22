package handlers

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	Config "github.com/m00lecule/stateful-scaling/config"
	Models "github.com/m00lecule/stateful-scaling/models"
	"github.com/orian/counters/global"
	Log "github.com/sirupsen/logrus"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
)

var sessionMuxKey = "sessions"

var cartMux = map[string]*sync.Mutex{}

var mockedData = RandomString(2)

func RandomString(n int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func CreateCart(c *gin.Context) {
	counter := global.GetCounter("app")
	counter.Increment()

	id := counter.Value()
	idStr := strconv.FormatInt(id, 10)

	cartMux[idStr] = &sync.Mutex{}

	bytes, _ := json.Marshal(map[string]string{})

	Config.RDB.Set(c.Request.Context(), idStr, bytes, 0).Err()
	Config.RDB.Do(c.Request.Context(), "EXPIRE", id, Config.Redis.TTL).Err()
	Config.RDB.SAdd(c.Request.Context(), sessionMuxKey, idStr).Err()

	c.JSON(http.StatusOK, gin.H{
		"payload":  idStr,
		"metadata": Config.Meta,
	})
}

func UpdateCart(c *gin.Context) {
	var cartUpdate Models.CartUpdate
	var cart = make(map[string]Models.ProductDetails)

	err := c.BindJSON(&cartUpdate)

	if err != nil {
		Log.Error("Could not unmarshall data")
		c.JSON(http.StatusInternalServerError, gin.H{"Error": err, "metadata": Config.Meta})
		return
	}

	Log.Info(cartUpdate.Details)

	id := c.Param("id")

	mx := cartMux[id]

	mx.Lock()

	currentCartBytes, err := Config.RDB.Get(c.Request.Context(), id).Result()

	err = json.Unmarshal([]byte(currentCartBytes), &cart)

	for id, delta := range cartUpdate.Details {
		Log.Debug(id, " - ", delta)
		if value, ok := cart[id]; ok {
			value.Count = value.Count + delta

			data := []string{}

			for i := uint(0); i < value.Count; i++ {
				data = append(data, mockedData)
			}

			value.Data = data
			cart[id] = value
		} else {
			data := []string{}

			for i := uint(0); i < delta; i++ {
				data = append(data, mockedData)
			}

			cart[id] = Models.ProductDetails{Count: delta, Data: data}
		}
	}

	bytes, err := json.Marshal(cart)

	Log.Info()
	err = Config.RDB.Set(c.Request.Context(), id, bytes, 0).Err()

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

	id := c.Param("id")

	cartDetailsBytes, err := Config.RDB.Get(c.Request.Context(), id).Result()

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
	var p Models.Product
	var cartDetails = make(map[string]Models.ProductDetails)

	id := c.Param("id")
	mx := cartMux[id]

	mx.Lock()

	cartDetailsBytes, err := Config.RDB.Get(c.Request.Context(), id).Result()

	err = json.Unmarshal([]byte(cartDetailsBytes), &cartDetails)

	if err != nil {
		mx.Unlock()
		c.JSON(http.StatusInternalServerError, gin.H{"Error": err})
		return
	}

	Log.Info(cartDetails)

	tx := Config.DB.Begin()

	for k, v := range cartDetails {
		if err = tx.Where("id = ?", k).First(&p).Error; err != nil {
			panic(err)
		}

		p.Stock -= v.Count

		if err = tx.Save(p).Error; err != nil {
			panic(err)
		}
	}

	tx.Commit()

	_, err = Config.RDB.Do(c.Request.Context(), "DEL", id).Result()

	if err != nil {
		mx.Unlock()
		c.JSON(http.StatusInternalServerError, gin.H{"Error": err,
			"metadata": Config.Meta,
		})
		return
	}

	mx.Unlock()

	delete(cartMux, id)
	err = Config.RDB.SRem(c.Request.Context(), sessionMuxKey, id).Err()

	c.JSON(http.StatusOK, gin.H{
		"metadata": Config.Meta,
	})
}
