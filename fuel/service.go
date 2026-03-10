package fuel

import (
	"context"
	"log"
	"quotes/fuel/sources"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// Orchestrator manages the fuel price collection process
type Orchestrator struct {
	config  *FuelConfig
	db      *DB
	sources map[string]sources.Source
}

func NewOrchestrator(config *FuelConfig, client *mongo.Client, dbName string) *Orchestrator {
	orc := &Orchestrator{
		config:  config,
		db:      NewDB(client, dbName),
		sources: make(map[string]sources.Source),
	}

	// Register sources
	orc.RegisterSource(sources.NewEUWeeklyOilBulletin())

	return orc
}

func (o *Orchestrator) RegisterSource(s sources.Source) {
	o.sources[s.Name()] = s
}

// Run executes the update process for all enabled countries
func (o *Orchestrator) Run(ctx context.Context) {
	for _, country := range o.config.Countries {
		if !country.Enabled {
			continue
		}

		source, ok := o.sources[country.Source]
		if !ok {
			log.Printf("Source %s not found for country %s", country.Source, country.Name)
			continue
		}

		if srcCfg, ok := o.config.Sources[country.Source]; !ok || !srcCfg.Enabled {
			continue
		}

		log.Printf("Fetching fuel prices for %s from %s", country.Name, source.Name())

		rawPrices, err := source.FetchPrices(ctx, country.Code)
		if err != nil {
			log.Printf("Error fetching prices for %s: %v", country.Name, err)
			continue
		}

		for _, raw := range rawPrices {
			priceUSD := raw.Price
			if raw.Currency != "USD" {
				// We want the final price in USD.
				// Based on DB inspection, EUR is the base (rate = 1).
				// USD rate is ~1.16 (EUR/USD).

				// Formula: PriceUSD = PriceCurrent / RateCurrent * RateUSD
				// If Current is EUR: PriceUSD = PriceEUR / 1 * RateUSD = PriceEUR * RateUSD

				rateUSD, err := o.db.GetRate(ctx, "USD")
				if err == nil && rateUSD > 0 {
					if raw.Currency == "EUR" {
						priceUSD = raw.Price * rateUSD
						log.Printf("Converted %s price %f EUR to %f USD (rate %f)", raw.Country, raw.Price, priceUSD, rateUSD)
					} else {
						// For other currencies, we'd need their rate too
						rateCurrent, err := o.db.GetRate(ctx, raw.Currency)
						if err == nil && rateCurrent > 0 {
							priceUSD = raw.Price / rateCurrent * rateUSD
						}
					}
				} else {
					log.Printf("Warning: could not get USD rate for conversion, storing raw price as USD")
				}
			}

			fp := FuelPrice{
				Country:      raw.Country,
				FuelType:     raw.FuelType,
				PriceUSD:     priceUSD,
				PriceLocal:   raw.Price,
				Source:       source.Name(),
				UpdatedAt:    time.Now(),
				SourceUpdate: raw.SourceUpdate,
			}

			if err := o.db.SavePrice(ctx, fp); err != nil {
				log.Printf("Error saving price for %s: %v", country.Name, err)
				continue
			}

			// Add to history
			history := FuelHistory{
				Country:    fp.Country,
				FuelType:   fp.FuelType,
				Date:       fp.SourceUpdate,
				PriceUSD:   fp.PriceUSD,
				PriceLocal: fp.PriceLocal,
			}

			if err := o.db.SaveHistory(ctx, history); err != nil {
				log.Printf("Error saving history for %s: %v", country.Name, err)
			}
		}
	}
}

// StartPeriodicUpdates starts a background goroutine for periodic updates
func (o *Orchestrator) StartPeriodicUpdates(ctx context.Context) {
	interval, err := time.ParseDuration(o.config.UpdateInterval)
	if err != nil {
		interval = 24 * time.Hour
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				o.Run(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}
