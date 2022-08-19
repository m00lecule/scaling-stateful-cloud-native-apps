package models

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"

	config "github.com/m00lecule/stateful-scaling/config"
	log "github.com/sirupsen/logrus"
)

const (
	cartsTableName = "carts"
)

type CartDetails struct {
	CartID    uint `gorm:"primaryKey"`
	ProductID uint `gorm:"primaryKey"`
	Count     uint `gorm:"not null; default:0"`
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
	Products    []Product                 `gorm:"many2many:cart_details;"`
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
	if !config.Meta.IsStateful {
		log.Info("Skipping carts offload - application is stateless")
		return nil
	}

	iter := config.RDB.Scan(ctx, 0, "*", 0).Iterator()

	var ids []uint
	var idsStr []string

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

		idsStr = append(idsStr, idStr)

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

	log.Info(fmt.Sprintf("Will try to offload %d carts", len(ids)))

	if len(ids) > 0 {
		config.DB.Table(cartsTableName).Where(ids).Updates(Cart{IsOrphan: true})

		log.Info("Carts offload done")
	}

	for _, sid := range idsStr {
		log.Info(fmt.Sprintf("Will try to offload cart %s", sid))

		if productsDetails, err := GetRedisCartDetails(ctx, sid, config.Tracer); err == nil {
			log.Info(fmt.Sprintf("Will try to offload %d products for cart %s", len(productsDetails), sid))

			for k, pd := range productsDetails {

				log.Info(fmt.Sprintf("Will try to offload product %s for cart %s", k, sid))

				pu64, err := strconv.ParseUint(k, 10, 32)

				if err != nil {
					log.Error(err)
					continue
				}

				cu64, err := strconv.ParseUint(sid, 10, 32)

				if err != nil {
					log.Error(err)
					continue
				}

				productID := uint(pu64)
				cartID := uint(cu64)

				if dbc := config.DB.Save(&CartDetails{CartID: cartID, ProductID: productID, Count: pd.Count}); dbc.Error != nil {
					log.Error("Had issues during data perstance during graceful shutdown")
					return nil
				}
			}
		} else {
			log.Error(err)
			return nil
		}
	}
	return nil
}

func ReadCartMux(ctx context.Context, idStr string, tracer trace.Tracer) *sync.Mutex {
	if mx := config.GetMux(idStr); mx == nil {
		log.Info(fmt.Sprintf("Will try to onload cart %s", idStr))
		if cart, err := TakeOverPostgresCart(ctx, idStr, tracer); err == nil && cart != nil {
			InitCart(ctx, cart)
			log.Info(fmt.Sprintf("Onloaded cart %s", idStr))
			return config.GetMux(idStr)
		}
	} else {
		return mx
	}
	return nil
}

func initCarts() {
	if !config.Meta.IsStateful {
		log.Info("Skipping carts offload - application is stateless")
		return
	}

	var carts []Cart

	query := &Cart{IsOrphan: true, IsSubmitted: false, OwnedBy: config.Meta.Hostname}

	if err := config.DB.Where(query).Find(&carts).Error; err != nil {
		panic(err)
	}

	log.Info(fmt.Sprintf("Found %d orphaned carts", len(carts)))

	for _, c := range carts {
		if err := InitCart(context.Background(), &c); err != nil {
			panic(err)
		}
	}

	log.Info(fmt.Sprintf("Initialized cartMux with %d entries", len(config.CartMux)))
}

func InitCart(ctx context.Context, c *Cart) error {
	id := strconv.FormatUint(uint64(c.ID), 10)

	config.InitMux(id)

	mx := config.GetMux(id)

	mx.Lock()
	defer mx.Unlock()

	c.IsOrphan = false

	if err := config.DB.Save(c).Error; err != nil {
		return err
	}

	var cartDetails []CartDetails

	if err := config.DB.Where("cart_id = ?", c.ID).Find(&cartDetails).Error; err != nil {
		return err
	}

	var cartProductDetails = make(map[string]ProductDetails)

	for _, cd := range cartDetails {
		cartProductDetails[strconv.FormatUint(uint64(cd.ProductID), 10)] = ProductDetails{Count: cd.Count, Data: []string{config.MockedData}}
	}

	bytes, _ := json.Marshal(cartProductDetails)

	sec, _ := time.ParseDuration(config.Redis.TTL)

	if err := config.RDB.Set(ctx, id, bytes, sec).Err(); err != nil {
		return err
	}

	if err := config.DB.WithContext(ctx).Where("cart_id = ?", c.ID).Delete(&cartDetails).Error; err != nil {
		return err
	}

	return nil
}

func UpdateRedisCart(ctx context.Context, id string, cartUpdate CartUpdate, tracer trace.Tracer) error {
	if !config.Meta.IsStateful {
		return nil
	}
	var cart = make(map[string]ProductDetails)

	ctx, redisSpan := tracer.Start(ctx, "redis")
	defer redisSpan.End()

	currentCartBytes, _ := config.RDB.Get(ctx, id).Result()

	if err := json.Unmarshal([]byte(currentCartBytes), &cart); err != nil || cart == nil {
		return err
	}

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

	if err := config.RDB.Set(ctx, id, bytes, -1).Err(); err != nil {
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

func TakeOverPostgresCart(ctx context.Context, cartIdStr string, tracer trace.Tracer) (*Cart, error) {
	c64, err := strconv.ParseUint(cartIdStr, 10, 32)

	if err != nil {
		return nil, err
	}

	cartId := uint(c64)

	ctx, postgresSpan := tracer.Start(ctx, "postgres")
	defer postgresSpan.End()

	tx := config.DB.Begin()

	cart := &Cart{}

	if dbc := tx.Where(&Cart{ID: cartId, IsOrphan: true, IsSubmitted: false}).Find(&cart); dbc.Error != nil || dbc.RowsAffected < 1 {
		tx.Rollback()
		return nil, dbc.Error
	}

	cart.OwnedBy = config.Meta.Hostname

	if dbc := tx.Save(&cart); dbc.Error != nil {
		tx.Rollback()
		return nil, dbc.Error
	}

	tx.Commit()

	return cart, nil
}

func GetRedisCartDetails(ctx context.Context, idStr string, tracer trace.Tracer) (map[string]ProductDetails, error) {
	if !config.Meta.IsStateful {
		return nil, nil
	}

	var cartDetails = make(map[string]ProductDetails)

	ctx, redisSpan := tracer.Start(ctx, "redis")
	defer redisSpan.End()

	cartDetailsBytes, err := config.RDB.Get(ctx, idStr).Result()

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(cartDetailsBytes), &cartDetails)

	if err != nil {
		return nil, err
	}

	return cartDetails, nil
}

func GetPostgresCartDetails(ctx context.Context, cartIdStr string, tracer trace.Tracer) (map[string]ProductDetails, error) {
	if config.Meta.IsStateful {
		return nil, nil
	}

	var productDetails = make(map[string]ProductDetails)

	ctx, postgresSpan := tracer.Start(ctx, "postgres")
	defer postgresSpan.End()

	c64, err := strconv.ParseUint(cartIdStr, 10, 32)

	if err != nil {
		return nil, err
	}

	cartId := uint(c64)
	cartDetails := []CartDetails{}

	if dbc := config.DB.WithContext(ctx).Where(&CartDetails{CartID: cartId}).Find(&cartDetails); dbc.Error != nil {
		return nil, dbc.Error
	}

	for i := 0; i < len(cartDetails); i += 1 {
		pId := strconv.FormatUint(uint64(cartDetails[i].ProductID), 10)
		productDetails[pId] = ProductDetails{Count: cartDetails[i].Count}
	}

	return productDetails, nil
}
