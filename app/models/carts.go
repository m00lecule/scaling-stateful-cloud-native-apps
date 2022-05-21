package models

type ProductDetails struct {
	Count uint
	Data  []string
}

type CartUpdate struct {
	Details map[string]uint `json:"details" binding:"required"`
}

type Cart struct {
	ID      string
	Content map[string]ProductDetails
}
