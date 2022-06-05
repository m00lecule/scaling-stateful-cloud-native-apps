package models

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
}
