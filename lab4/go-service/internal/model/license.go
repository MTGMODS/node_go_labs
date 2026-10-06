package model

type License struct {
	ID           int    `json:"id"`
	Key          string `json:"key"`
	Product      string `json:"product"`
	Owner        string `json:"owner"`
	Status       string `json:"status"`
	DurationDays int    `json:"duration_days"`
	MaxDevices   int    `json:"max_devices"`
}

type LicenseInput struct {
	Key          *string `json:"key"`
	Product      *string `json:"product"`
	Owner        *string `json:"owner"`
	Status       *string `json:"status"`
	DurationDays *int    `json:"duration_days"`
	MaxDevices   *int    `json:"max_devices"`
}
