package fuel

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetPrices returns the latest fuel prices for a country
func (d *DB) GetPrices(ctx context.Context, country string) ([]FuelPrice, error) {
	collection := d.client.Database(d.dbName).Collection("FuelPrices")

	filter := bson.M{"country": country}
	if country == "" {
		filter = bson.M{}
	}

	cur, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var prices []FuelPrice
	if err := cur.All(ctx, &prices); err != nil {
		return nil, err
	}
	return prices, nil
}

// GetHistory returns historical fuel prices for a country and fuel type
func (d *DB) GetHistory(ctx context.Context, country, fuelType string, limit int) ([]FuelHistory, error) {
	collection := d.client.Database(d.dbName).Collection("FuelHistory")

	filter := bson.M{
		"country":   country,
		"fuel_type": fuelType,
	}

	opts := options.Find().SetLimit(int64(limit)).SetSort(bson.M{"date": -1})
	cur, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var history []FuelHistory
	if err := cur.All(ctx, &history); err != nil {
		return nil, err
	}
	return history, nil
}
