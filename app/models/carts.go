package models

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	config "github.com/m00lecule/stateful-scaling/config"
	log "github.com/sirupsen/logrus"
)

type ProductDetails struct {
	Count uint
	Data  []string
}

type CartUpdate struct {
	Details map[string]uint `json:"details" binding:"required"`
}

type Cart struct {
	ID          uint                      `gorm:"primaryKey"`
	Content     map[string]ProductDetails `gorm:"-"`
	IsOrphan    bool                      `gorm:"not null; type:boolean; default:false"`
	IsSubmitted bool                      `gorm:"not null; type:boolean; default:false"`
	OwnedBy     string                    `gorm:"not null; type:varchar(16); default:false"`
	CreatedAt   time.Time
}

var (
	cartsTableName = "carts"
)

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
