package fuel

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// DB handles fuel-related database operations
type DB struct {
	client *mongo.Client
	dbName string
}

func NewDB(client *mongo.Client, dbName string) *DB {
	return &DB{
		client: client,
		dbName: dbName,
	}
}

// DeletePricesForCountrySource removes all fuel price documents for the given country and source.
// Call before saving a full batch to avoid duplicates when renaming fuel types.
func (d *DB) DeletePricesForCountrySource(ctx context.Context, country, source string) error {
	collection := d.client.Database(d.dbName).Collection("FuelPrices")
	_, err := collection.DeleteMany(ctx, bson.M{"country": country, "source": source})
	return err
}

// SavePrice updates the current fuel price in the database
func (d *DB) SavePrice(ctx context.Context, price FuelPrice) error {
	collection := d.client.Database(d.dbName).Collection("FuelPrices")

	filter := bson.M{
		"country":   price.Country,
		"fuel_type": price.FuelType,
	}

	update := bson.M{
		"$set": price,
	}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// SaveHistory saves a historical data point, updating if it exists for the same day
func (d *DB) SaveHistory(ctx context.Context, history FuelHistory) error {
	collection := d.client.Database(d.dbName).Collection("FuelHistory")

	// Ensure the date is truncated to the day
	history.Date = time.Date(history.Date.Year(), history.Date.Month(), history.Date.Day(), 0, 0, 0, 0, time.UTC)

	filter := bson.M{
		"country":   history.Country,
		"fuel_type": history.FuelType,
		"date":      history.Date,
	}

	update := bson.M{
		"$set": history,
	}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// GetRate returns the rate for a specific currency symbol from the main Quotes collection
func (d *DB) GetRate(ctx context.Context, symbol string) (float64, error) {
	// Our main database is Quotes, collection Currencies
	// According to dbconnection.go: client.Database("Quotes").Collection("Currencies")
	collection := d.client.Database("Quotes").Collection("Currencies")

	var result struct {
		Symbol string  `bson:"symbol"`
		Rate   float64 `bson:"rate"`
	}

	filter := bson.M{"symbol": symbol}
	err := collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		return 0, err
	}

	return result.Rate, nil
}
