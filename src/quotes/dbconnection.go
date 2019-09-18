package main

import (
	"context"
	"fmt"
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
	client.Disconnect(ctx)
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
	client.Disconnect(ctx)
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
	var result []*HistoryQuote
	fmt.Printf("sdasd")
	var date = time.Now().Format("01-02-2006")
	client := dbConnect()
	collection := client.Database("Quotes").Collection("History")
	filter := bson.D{
		{"symbol", currency.Symbol},
		{"category", currency.Category},
		{"date", date},
	}

	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	cur, err := collection.Find(ctx, filter)
	if err != nil {
		log.Fatal(err)
	}
	client.Disconnect(ctx)

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
	cur.Close(ctx)
	fmt.Printf("%v", len(result))
	if len(result) >= 1 {
		Updatehistory(currency)
		return
	}
	if len(result) < 1 {
		fmt.Print("%v", len(result))
		AddHistory(currency)
		return
	}
}

// AddHistory - add new history record
func AddHistory(currency Quote) {
	client := dbConnect()
	var h HistoryQuote
	h.Category = currency.Category
	h.Date = time.Now().Format("01-02-2006")
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
	client := dbConnect()
	collection := client.Database("Quotes").Collection("History")
	filter := bson.D{
		{"symbol", currency.Symbol},
		{"category", currency.Category},
		{"date", date},
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
