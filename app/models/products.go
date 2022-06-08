package models

import (
	config "github.com/m00lecule/stateful-scaling/config"
)

type Product struct {
	ID    int    `gorm:"primaryKey"`
	Name  string `gorm:"unique; not null; type:varchar(100); default:null" json:"Name" binding:"required"`
	Stock uint   `gorm:"check:possitive_stock,stock >= 0" json:"Stock"`
}

func AddProduct(p *Product) (err error) {
	if err = config.DB.Create(p).Error; err != nil {
		return err
	}
	return nil
}

func DelProduct(p *Product) (err error) {
	if err = config.DB.Where("product_id = ?", p.ID).Delete(&CartDetails{}).Error; err != nil {
		return err
	}

	if err = config.DB.Delete(p).Error; err != nil {
		return err
	}
	return nil
}

func GetOneProduct(p *Product, id int) (err error) {
	if err := config.DB.Where("id = ?", id).First(p).Error; err != nil {
		return err
	}
	return nil
}

func (b *Product) TableName() string {
	return "products"
}
