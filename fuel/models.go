package fuel

import (
	"time"
)

// FuelPrice represents the current price of a fuel type in a country
type FuelPrice struct {
	Country      string    `json:"country" bson:"country"`
	FuelType     string    `json:"fuel_type" bson:"fuel_type"`
	PriceUSD     float64   `json:"price_usd" bson:"price_usd"`
	PriceLocal   float64   `json:"price_local" bson:"price_local"`
	Source       string    `json:"source" bson:"source"`
	UpdatedAt    time.Time `json:"updated_at" bson:"updated_at"`
	SourceUpdate time.Time `json:"source_update" bson:"source_update"`
}

// FuelHistory represents a historical price point for a daily interval
type FuelHistory struct {
	Country    string    `json:"country" bson:"country"`
	FuelType   string    `json:"fuel_type" bson:"fuel_type"`
	Date       time.Time `json:"date" bson:"date"` // Truncated to the day
	PriceUSD   float64   `json:"price_usd" bson:"price_usd"`
	PriceLocal float64   `json:"price_local" bson:"price_local"`
}
