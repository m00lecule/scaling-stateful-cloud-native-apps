package models

import (
	Config "github.com/m00lecule/stateful-scaling/config"
)

type Product struct {
	ID    int    `gorm:"primaryKey"`
	Name  string `gorm:"unique;not null;type:varchar(100);default:null" json:"Name" binding:"required"`
	Stock uint   `gorm:"check:possitive_stock,stock >= 0" json:"Stock"`
}

func MigrateProducts() {
	var DB = Config.DB
	DB.AutoMigrate(&Product{})
	DB.Migrator().CreateConstraint(&Product{}, "possitive_stock")
}

func AddProduct(p *Product) (err error) {
	if err = Config.DB.Create(p).Error; err != nil {
		return err
	}
	return nil
}

func DelProduct(p *Product) (err error) {
	if err = Config.DB.Delete(p).Error; err != nil {
		return err
	}
	return nil
}

func GetOneProduct(p *Product, id int) (err error) {
	if err := Config.DB.Where("id = ?", id).First(p).Error; err != nil {
		return err
	}
	return nil
}

func (b *Product) TableName() string {
	return "products"
}
