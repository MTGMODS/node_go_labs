package model

type License struct {
	ID           int    `json:"id" bson:"_id"`
	Key          string `json:"key" bson:"key"`
	Product      string `json:"product" bson:"product"`
	Owner        string `json:"owner" bson:"owner"`
	Status       string `json:"status" bson:"status"`
	DurationDays int    `json:"duration_days" bson:"duration_days"`
	MaxDevices   int    `json:"max_devices" bson:"max_devices"`
}

type LicenseInput struct {
	Key          *string `json:"key"`
	Product      *string `json:"product"`
	Owner        *string `json:"owner"`
	Status       *string `json:"status"`
	DurationDays *int    `json:"duration_days"`
	MaxDevices   *int    `json:"max_devices"`
}
