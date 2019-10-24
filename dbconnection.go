package main

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func dbConnect() *mongo.Client {
	client, err := mongo.NewClient(options.Client().ApplyURI(Config.Hosts.Mongodb))
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	if err = client.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	return client
}

func getAllElementsinMemory() {
	client := dbConnect()
	collection := client.Database("Quotes").Collection("Currencies")
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	cur, err := collection.Find(ctx, bson.D{})
	if err != nil {
		log.Fatal(err)
	}
	for cur.Next(ctx) {
		var elem Quote
		err := cur.Decode(&elem)
		if err != nil {
			log.Fatal(err)
		}
		var present = false
		for index := range QutesinMemory {
			if QutesinMemory[index].Category == elem.Category {
				if QutesinMemory[index].Symbol == elem.Symbol {
					QutesinMemory[index].Rate = elem.Rate
					present = true
				}
			}
		}
		if !(present) {
			QutesinMemory = append(QutesinMemory, &elem)
		}
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}
	client.Disconnect(ctx)
	cur.Close(ctx)
}

func isElementInDB(currency Quote) bool {
	var result []*Quote
	client := dbConnect()
	collection := client.Database("Quotes").Collection("Currencies")
	filter := bson.D{
		{"symbol", currency.Symbol},
		{"category", currency.Category},
	}
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	cur, err := collection.Find(ctx, filter)
	if err != nil {
		log.Fatal(err)
	}
	for cur.Next(ctx) {
		var elem Quote
		err := cur.Decode(&elem)
		if err != nil {
			log.Fatal(err)
		}
		result = append(result, &elem)
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}
	client.Disconnect(ctx)
	cur.Close(ctx)
	if len(result) >= 1 {
		return true
	}
	return false
}

func updateRate(currency Quote) {
	client := dbConnect()
	collection := client.Database("Quotes").Collection("Currencies")
	filter := bson.D{
		{"symbol", currency.Symbol},
		{"category", currency.Category},
	}
	update := bson.D{
		{"$set", bson.D{
			{"rate", currency.Rate},
		}},
	}
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Fatal(err)
	}
	client.Disconnect(ctx)
	writeHistory(currency)
}

func writeNewCurrency(currency Quote) {
	client := dbConnect()
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	collection := client.Database("Quotes").Collection("Currencies")
	_, err := collection.InsertOne(ctx, currency)
	if err != nil {
		log.Fatal(err)
	}
	client.Disconnect(ctx)
	writeHistory(currency)
}

// writeHistory - Working with history of quotes
func writeHistory(currency Quote) {
	var day = int(time.Now().Weekday())
	if currency.Category != 1 {
		if (day == 0) || (day == 6) {
			return
		}
	}
	var result []*HistoryQuote
	var date = time.Now().Format("01-02-2006")
	var layout = "01-02-2006"
	var d, _ = time.Parse(layout, date)
	client := dbConnect()
	collection := client.Database("Quotes").Collection("History")
	filter := bson.D{
		{"symbol", currency.Symbol},
		{"category", currency.Category},
		{"date", d},
	}

	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	cur, err := collection.Find(ctx, filter)
	if err != nil {
		log.Fatal(err)
	}

	for cur.Next(ctx) {
		var elem HistoryQuote
		err := cur.Decode(&elem)
		if err != nil {
			log.Fatal(err)
		}
		result = append(result, &elem)
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}
	client.Disconnect(ctx)
	cur.Close(ctx)
	if len(result) >= 1 {
		Updatehistory(currency)
		return
	}
	if len(result) < 1 {
		AddHistory(currency)
		return
	}
}

// AddHistory - add new history record
func AddHistory(currency Quote) {
	client := dbConnect()
	var h HistoryQuote
	h.Category = currency.Category
	var layout = "01-02-2006"
	var date = time.Now().Format("01-02-2006")
	h.Date, _ = time.Parse(layout, date)
	h.Rate = currency.Rate
	h.Symbol = currency.Symbol
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	collection := client.Database("Quotes").Collection("History")
	_, err := collection.InsertOne(ctx, h)
	if err != nil {
		log.Fatal(err)
	}
	client.Disconnect(ctx)
}

// Updatehistory - update existing history
func Updatehistory(currency Quote) {
	var date = time.Now().Format("01-02-2006")
	var layout = "01-02-2006"
	var d, _ = time.Parse(layout, date)
	client := dbConnect()
	collection := client.Database("Quotes").Collection("History")
	filter := bson.D{
		{"symbol", currency.Symbol},
		{"category", currency.Category},
		{"date", d},
	}
	update := bson.D{
		{"$set", bson.D{
			{"rate", currency.Rate},
		}},
	}
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Fatal(err)
	}
	client.Disconnect(ctx)
}

func loadHistory(s string, c int, t int) map[string]float64 {
	var r = make(map[string]float64)

	client := dbConnect()
	collection := client.Database("Quotes").Collection("History")
	filter := bson.D{
		{"symbol", s},
		{"category", c},
	}
	options := options.Find()
	options.SetLimit(int64(t))
	options.SetSort(bson.D{{"date", -1}})
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	cur, err := collection.Find(ctx, filter, options)
	if err != nil {
		log.Fatal(err)
	}

	for cur.Next(ctx) {
		var elem HistoryQuote
		err := cur.Decode(&elem)
		if err != nil {
			log.Fatal(err)
		}
		r[elem.Date.Format("2006-01-02")] = elem.Rate
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}
	client.Disconnect(ctx)
	cur.Close(ctx)
	return r
}
