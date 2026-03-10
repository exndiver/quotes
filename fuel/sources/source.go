package sources

import (
	"context"
	"time"
)

// RawFuelPrice represents a raw price from a source before standardization
type RawFuelPrice struct {
	Country      string
	FuelType     string
	Price        float64
	Currency     string
	SourceUpdate time.Time
}

// Source is the interface all fuel price fetchers must implement
type Source interface {
	Name() string
	FetchPrices(ctx context.Context, countryCode string) ([]RawFuelPrice, error)
}
