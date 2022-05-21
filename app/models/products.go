package models

import (
	Config "github.com/m00lecule/stateful-scaling/config"
)

type Product struct {
	ID    int    `gorm:"primaryKey"`
	Name  string `json:"Name" binding:"required"`
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

func GetOneProduct(p *Product, id int) (err error) {
	if err := Config.DB.Where("id = ?", id).First(p).Error; err != nil {
		return err
	}
	return nil
}

// func PutOneProduct(p *Product, id string) (err error) {
// 	config.DB.Save(p)
// 	return nil
// }

// func DeleteProduct(p *Product, id string) (err error) {
// 	config.DB.Where("id = ?", id).Delete(p)
// 	return nil
// }

func (b *Product) TableName() string {
	return "products"
}
