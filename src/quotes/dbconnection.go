package main

import (
	"log"
	"context"
	"time"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/bson"
)

func db_connect() *mongo.Client{
	client, err := mongo.NewClient(options.Client().ApplyURI(Config.Hosts.Mongodb))
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	if err = client.Connect(ctx); err != nil {log.Fatal(err)}
	return client
}

func getAllElementsinMemory(){
	client := db_connect()
	collection := client.Database("Quotes").Collection("Currencies")
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	cur, err := collection.Find(ctx, bson.D{})
	if err != nil { log.Fatal(err) }
	client.Disconnect(ctx)
	for cur.Next(ctx) {
		var elem Quote
		err := cur.Decode(&elem)
		if err != nil {log.Fatal(err)}
		var present = false
		for index, _ := range QutesinMemory {
			if QutesinMemory[index].Category == elem.Category{
				if QutesinMemory[index].Symbol == elem.Symbol {
					QutesinMemory[index].Rate = elem.Rate
					present = true
				}
			}
		}
		if !(present) {QutesinMemory = append(QutesinMemory, &elem)}
	}
	if err := cur.Err(); err != nil {log.Fatal(err)}
	cur.Close(ctx)
}

func isElementInDB(currency Quote) bool {
	var result []* Quote
	client := db_connect()
	collection := client.Database("Quotes").Collection("Currencies")
	filter := bson.D{
		{"symbol",  currency.Symbol}, 
		{"category", currency.Category},
	}
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	cur, err := collection.Find(ctx, filter)
	if err != nil { log.Fatal(err) }
	client.Disconnect(ctx)
	for cur.Next(ctx) {
		var elem Quote
		err := cur.Decode(&elem)
		if err != nil {log.Fatal(err)}
		result = append(result, &elem)
	}
	if err := cur.Err(); err != nil {log.Fatal(err)}
	cur.Close(ctx)
	if len(result) >= 1 {
		return true
	}
	return false
}

func updateRate(currency Quote){
	client := db_connect()
	collection := client.Database("Quotes").Collection("Currencies")
	filter := bson.D{
		{"symbol",  currency.Symbol}, 
		{"category", currency.Category},
	}
	update := bson.D{
		{"$set", bson.D{
			{"rate", currency.Rate},
		}},
	}
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil { log.Fatal(err) }
	client.Disconnect(ctx)
}

func writeNewCurrency(currency Quote){
	client := db_connect()
	ctx,_:= context.WithTimeout(context.Background(), 30*time.Second)
	collection := client.Database("Quotes").Collection("Currencies")
	_, err := collection.InsertOne(ctx,currency)
	if err != nil { log.Fatal(err) }
	client.Disconnect(ctx)
}