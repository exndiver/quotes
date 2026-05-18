package fuel

import (
	"context"
	"fmt"
	"log"
	"quotes/fuel/sources"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// Orchestrator manages the fuel price collection process
type Orchestrator struct {
	config  *FuelConfig
	db      *DB
	sources map[string]sources.Source

	runMu           sync.RWMutex
	lastRunAt       time.Time
	lastRunFailures []string
	lastRunMessage  string
	updateInterval  time.Duration
}

func NewOrchestrator(config *FuelConfig, client *mongo.Client, dbName string) *Orchestrator {
	interval, err := time.ParseDuration(config.UpdateInterval)
	if err != nil || interval <= 0 {
		interval = 24 * time.Hour
	}
	orc := &Orchestrator{
		config:         config,
		db:             NewDB(client, dbName),
		sources:        make(map[string]sources.Source),
		updateInterval: interval,
	}

	// Register sources
	orc.RegisterSource(sources.NewEUWeeklyOilBulletin())
	orc.RegisterSource(sources.NewUkraineMinfin())
	orc.RegisterSource(sources.NewUKGov())
	orc.RegisterSource(sources.NewThailandPTTOR())
	orc.RegisterSource(sources.NewGeorgiaWissol())
	orc.RegisterSource(sources.NewRussiaGlobalPetrolPrices())
	orc.RegisterSource(sources.NewUSAGlobalPetrolPrices())
	orc.RegisterSource(sources.NewGPP())

	return orc
}

func (o *Orchestrator) RegisterSource(s sources.Source) {
	o.sources[s.Name()] = s
}

// MaxStaleAfter returns how long after last success the fuel module is considered stale.
func (o *Orchestrator) MaxStaleAfter() time.Duration {
	return o.updateInterval * 2
}

// LastRunFailures returns errors from the most recent Run.
func (o *Orchestrator) LastRunFailures() []string {
	o.runMu.RLock()
	defer o.runMu.RUnlock()
	out := make([]string, len(o.lastRunFailures))
	copy(out, o.lastRunFailures)
	return out
}

// LastRunMessage returns a short summary of the most recent Run.
func (o *Orchestrator) LastRunMessage() string {
	o.runMu.RLock()
	defer o.runMu.RUnlock()
	return o.lastRunMessage
}

// LastRunAt returns when Run last finished.
func (o *Orchestrator) LastRunAt() time.Time {
	o.runMu.RLock()
	defer o.runMu.RUnlock()
	return o.lastRunAt
}

func (o *Orchestrator) finishRun(successes int, failures []string) {
	o.runMu.Lock()
	defer o.runMu.Unlock()
	o.lastRunAt = time.Now()
	o.lastRunFailures = failures
	if len(failures) == 0 {
		o.lastRunMessage = fmt.Sprintf("updated %d countries", successes)
		return
	}
	o.lastRunMessage = fmt.Sprintf("updated %d countries, %d failed", successes, len(failures))
}

// Run executes the update process for all enabled countries
func (o *Orchestrator) Run(ctx context.Context) {
	var failures []string
	successes := 0
	defer func() {
		o.finishRun(successes, failures)
	}()

	for _, country := range o.config.Countries {
		if !country.Enabled {
			continue
		}

		source, ok := o.sources[country.Source]
		if !ok {
			msg := fmt.Sprintf("%s: source %s not found", country.Name, country.Source)
			log.Print(msg)
			failures = append(failures, msg)
			continue
		}

		if srcCfg, ok := o.config.Sources[country.Source]; !ok || !srcCfg.Enabled {
			continue
		}

		log.Printf("Fetching fuel prices for %s from %s", country.Name, source.Name())

		rawPrices, err := source.FetchPrices(ctx, country.Code)
		if err != nil {
			msg := fmt.Sprintf("%s (%s): %v", country.Name, source.Name(), err)
			log.Printf("Error fetching prices for %s: %v", country.Name, err)
			failures = append(failures, msg)
			continue
		}

		// Throttle: delay after each request for this source (e.g. GPP once per day spread)
		if srcCfg, ok := o.config.Sources[country.Source]; ok && srcCfg.RequestDelaySeconds > 0 {
			delay := time.Duration(srcCfg.RequestDelaySeconds) * time.Second
			log.Printf("Throttle: waiting %v before next %s request", delay, source.Name())
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return
			}
		}

		// Replace all prices for this country/source to avoid duplicates after renaming fuel types
		if err := o.db.DeletePricesForCountrySource(ctx, country.Code, source.Name()); err != nil {
			log.Printf("Warning: could not delete old prices for %s %s: %v", country.Code, source.Name(), err)
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
						// For other currencies (GBP, UAH, etc.) we need their rate
						rateCurrent, err := o.db.GetRate(ctx, raw.Currency)
						if err == nil && rateCurrent > 0 {
							priceUSD = raw.Price / rateCurrent * rateUSD
							log.Printf("Converted %s price %f %s to %f USD (rate %s=%f)", raw.Country, raw.Price, raw.Currency, priceUSD, raw.Currency, rateCurrent)
						} else {
							log.Printf("Warning: no rate for %s (%s), cannot convert %s price to USD", raw.Currency, raw.Country, raw.Currency)
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
		successes++
	}
}

// StartPeriodicUpdates starts a background goroutine for periodic updates.
// onComplete is called after each Run (e.g. to refresh health status).
func (o *Orchestrator) StartPeriodicUpdates(ctx context.Context, onComplete func()) {
	interval := o.updateInterval

	go func() {
		log.Printf("Fuel periodic updates enabled: interval=%v", interval)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				start := time.Now()
				log.Printf("Fuel periodic update started")
				o.Run(ctx)
				if onComplete != nil {
					onComplete()
				}
				log.Printf("Fuel periodic update finished in %v", time.Since(start))
			case <-ctx.Done():
				return
			}
		}
	}()
}
