package models

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"
	"encoding/json"

	"go.opentelemetry.io/otel/trace"
	
	config "github.com/m00lecule/stateful-scaling/config"
	log "github.com/sirupsen/logrus"
)

const (
	cartsTableName = "carts"
)

type CartDetails struct {
	CartID      uint `gorm:"primaryKey"`
	ProductID	uint `gorm:"primaryKey"`
	Count		uint `gorm:"not null; default:"0"`
}

type ProductDetails struct {
	Count uint
	Data  []string
}

type CartUpdate struct {
	Details map[string]uint `json:"details" binding:"required"`
}

type Cart struct {
	ID          uint                      `gorm:"primaryKey"`
	Products 	[]Product				  `gorm:"many2many:cart_details;"`	
	Content     map[string]ProductDetails `gorm:"-"`
	IsOrphan    bool                      `gorm:"not null; type:boolean; default:false"`
	IsSubmitted bool                      `gorm:"not null; type:boolean; default:false"`
	OwnedBy     string                    `gorm:"not null; type:varchar(16); default:false"`
	CreatedAt   time.Time
}

func (b *Cart) TableName() string {
	return cartsTableName
}

func offloadCarts(ctx context.Context) (err error) {
	iter := config.RDB.Scan(ctx, 0, "*", 0).Iterator()

	var ids []uint

	for iter.Next(ctx) {
		idStr := iter.Val()

		if idStr == config.Meta.SessionMuxKey {
			log.Debug(fmt.Sprintf("Skipping %s", idStr))
			continue
		}

		ttl, err := config.RDB.TTL(ctx, idStr).Result()

		if err != nil {
			log.Error(fmt.Sprintf("Skipping %s", idStr))
			log.Error(err)
			continue
		}

		if ttl <= 0 {
			log.Info(fmt.Sprintf("Skipping %s - expired", idStr))
			continue
		}

		u64, err := strconv.ParseUint(idStr, 10, 32)

		if err != nil {
			log.Error(err)
			continue
		}

		log.Info(fmt.Sprintf("Offlading session: %s", idStr))

		id := uint(u64)

		ids = append(ids, id)
	}

	if err := iter.Err(); err != nil {
		panic(err)
	}

	log.Info(fmt.Sprintf("Will try to offload %d models", len(ids)))

	if len(ids) > 0 {
		config.DB.Table(cartsTableName).Where(ids).Updates(Cart{IsOrphan: true})
		log.Info("Carts offload done")
	}

	return nil
}

func initCarts() {
	var carts []Cart
	query := &Cart{IsOrphan: true, IsSubmitted: false, OwnedBy: config.Meta.Hostname}

	if err := config.DB.Where(query).Find(&carts).Error; err != nil {
		panic(err)
	}

	log.Info(fmt.Sprintf("Found %d orphaned carts", len(carts)))

	for _, c := range carts {
		log.Info(c.ID)

		id := strconv.FormatUint(uint64(c.ID), 10)
		config.CartMux[id] = &sync.Mutex{}

		c.IsOrphan = false

		if err := config.DB.Save(c).Error; err != nil {
			panic(err)
		}
	}

	log.Info(fmt.Sprintf("Initialized cartMux with %d entries", len(config.CartMux)))
}

func UpdateRedisCart(ctx context.Context, id string, cartUpdate CartUpdate, tracer trace.Tracer) error {			
	if ! config.Meta.IsStateful {
		return nil
	}
	
	var cart = make(map[string]ProductDetails)

	ctx, redisSpan := tracer.Start(ctx, "redis")
	defer redisSpan.End()

	currentCartBytes, _ := config.RDB.Get(ctx, id).Result()

	_ = json.Unmarshal([]byte(currentCartBytes), &cart)

	for id, delta := range cartUpdate.Details {
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

			cart[id] = ProductDetails{Count: delta, Data: data}
		}
	}

	bytes, _ := json.Marshal(cart)

	if err := config.RDB.Set(ctx, id, bytes, 0).Err(); err != nil {
		return err
	}
	
	return nil
}

func UpdatePostgresCart(ctx context.Context, cartIdStr string, cartUpdate CartUpdate, tracer trace.Tracer) error {			
	if config.Meta.IsStateful {
		return nil
	}

	c64, err := strconv.ParseUint(cartIdStr, 10, 32)

	if err != nil {
		return err
	}
	cartId := uint(c64)
	
	ctx, postgresSpan := tracer.Start(ctx, "postgres")
	defer postgresSpan.End()

	tx := config.DB.Begin()

	for prodIdStr, delta := range cartUpdate.Details {
		u64, err := strconv.ParseUint(prodIdStr, 10, 32)
		if err != nil {
			tx.Rollback()
			return err
		}
		prodId := uint(u64)

		cartDetails := &CartDetails{ProductID: prodId, CartID: cartId}
		
		if dbc := tx.Limit(1).Find(&cartDetails); dbc.Error != nil {
			tx.Rollback()
			return dbc.Error
		} else {
			if dbc.RowsAffected > 0 {
				cartDetails.Count = cartDetails.Count + delta
			} else {
				cartDetails.Count = delta
			}
		}

		if dbc := tx.Save(&cartDetails); dbc.Error != nil {
			tx.Rollback()
			return dbc.Error
		}
	}
	tx.Commit()
	
	return nil
}
