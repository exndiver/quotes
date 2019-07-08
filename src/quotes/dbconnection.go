package main

import (
	"log"
	"context"
	"strings"
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

func getAllElements() []*Quote {
	var result []* Quote
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
		result = append(result, &elem)
	}
	if err := cur.Err(); err != nil {log.Fatal(err)}
	cur.Close(ctx)
	return result
}

func getOneGroup(Category int) []*Quote {
	var result []* Quote
	client := db_connect()
	collection := client.Database("Quotes").Collection("Currencies")
	filter := bson.D{
		{"category", Category},
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
	return result
}


func getOneElement(Category int, Symbol string) Quote {
	var result Quote
	client := db_connect()
	collection := client.Database("Quotes").Collection("Currencies")
	filter := bson.D{
		{"category", Category},
		{"symbol", strings.ToUpper(Symbol)},
	}
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	err := collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		log.Panic(err)
	}
	client.Disconnect(ctx)
	return result
}