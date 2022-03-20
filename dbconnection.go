package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// dbConnect - opens the connection to the DB
func dbConnect() *mongo.Client {
	clientOptions := options.Client().ApplyURI(Config.Hosts.Mongodb)
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		logError("DB problem!", err.Error(), 2)
		log.Fatalf("DB problem!")
	}
	// Check the connection
	err = client.Ping(context.TODO(), nil)

	if err != nil {
		logError("DB problem!", err.Error(), 2)
		log.Fatalf("DB problem!")
	}

	fmt.Println("Connected to MongoDB!")
	return client
}

func getAllElementsinMemory() {
	errPing := client.Ping(context.TODO(), nil)
	if errPing != nil {
		client = dbConnect()
		logError("getAllElementsinMemory - DB connection is lost!", errPing.Error(), 3)
	}
	collection := client.Database("Quotes").Collection("Currencies")
	cur, err := collection.Find(context.TODO(), bson.D{})
	if err != nil {
		logError("DB problem!", err.Error(), 2)
		log.Fatalf("DB problem!")
	}
	for cur.Next(context.TODO()) {
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
		logError("DB problem!", err.Error(), 2)
		log.Fatalf("DB problem!")
	}

	cur.Close(context.TODO())
}

func isElementInDB(currency Quote) bool {
	var result []*Quote
	errPing := client.Ping(context.TODO(), nil)
	if errPing != nil {
		client = dbConnect()
		logError("isElementInDB - DB connection is lost!", errPing.Error(), 3)
	}
	collection := client.Database("Quotes").Collection("Currencies")
	filter := bson.D{
		primitive.E{Key: "symbol", Value: currency.Symbol},
		primitive.E{Key: "category", Value: currency.Category},
	}
	cur, err := collection.Find(context.TODO(), filter)
	if err != nil {
		logError("DB problem!", err.Error(), 2)
		log.Fatalf("DB problem!")
	}
	for cur.Next(context.TODO()) {
		var elem Quote
		err := cur.Decode(&elem)
		if err != nil {
			logError("DB problem!", err.Error(), 2)
			log.Fatalf("DB problem!")
		}
		result = append(result, &elem)
	}
	if err := cur.Err(); err != nil {
		logError("DB problem!", err.Error(), 2)
		log.Fatalf("DB problem!")
	}

	cur.Close(context.TODO())

	if len(result) > 1 {
		_, errDel := collection.DeleteMany(context.TODO(), filter)
		logError("More than one in Quotes!", currency.Symbol, 3)
		if errDel != nil {
			logError("DB problem!", errDel.Error(), 2)
			log.Fatalf("DB problem!")
		}
		return false
	}

	if len(result) == 1 {
		return true
	}
	return false
}

func updateRate(currency Quote) {
	start := time.Now()
	errPing := client.Ping(context.TODO(), nil)
	if errPing != nil {
		client = dbConnect()
		logError("updateRate - DB connection is lost!", errPing.Error(), 3)
	}
	collection := client.Database("Quotes").Collection("Currencies")
	filter := bson.D{
		primitive.E{Key: "symbol", Value: currency.Symbol},
		primitive.E{Key: "category", Value: currency.Category},
	}
	update := bson.D{
		primitive.E{Key: "$set", Value: bson.D{
			primitive.E{Key: "rate", Value: currency.Rate},
		}},
	}
	_, err := collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		logError("DB problem!", err.Error(), 2)
		log.Fatalf("DB problem!")
	}
	elapsed := int64(time.Since(start) / time.Millisecond)
	if Config.LogDebug {
		logEvent(7, "Rate update", 200, currency.Symbol, elapsed)
	}
//	writeHistory(currency)
}

func writeNewCurrency(currency Quote) {
	start := time.Now()
	errPing := client.Ping(context.TODO(), nil)
	if errPing != nil {
		client = dbConnect()
		logError("writeNewCurrency - DB connection is lost!", errPing.Error(), 3)
	}
	collection := client.Database("Quotes").Collection("Currencies")
	_, err := collection.InsertOne(context.TODO(), currency)
	if err != nil {
		logError("DB problem!", err.Error(), 2)
		log.Fatalf("DB problem!")
	}
	elapsed := int64(time.Since(start) / time.Millisecond)
	if Config.LogDebug {
		logEvent(7, "Rate writed", 200, currency.Symbol, elapsed)
	}
//	writeHistory(currency)
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
	errPing := client.Ping(context.TODO(), nil)
	if errPing != nil {
		client = dbConnect()
		logError("writeHistory - DB connection is lost!", errPing.Error(), 3)
	}
	collection := client.Database("Quotes").Collection("History")
	filter := bson.D{
		primitive.E{Key: "symbol", Value: currency.Symbol},
		primitive.E{Key: "category", Value: currency.Category},
		primitive.E{Key: "date", Value: d},
	}

	cur, err := collection.Find(context.TODO(), filter)
	if err != nil {
		logError("DB problem!", err.Error(), 2)
		log.Fatalf("DB problem!")
	}

	for cur.Next(context.TODO()) {
		var elem HistoryQuote
		err := cur.Decode(&elem)
		if err != nil {
			logError("DB problem!", err.Error(), 2)
			log.Fatalf("DB problem!")
		}
		result = append(result, &elem)
	}
	if err := cur.Err(); err != nil {
		logError("DB problem!", err.Error(), 2)
		log.Fatalf("DB problem!")
	}

	cur.Close(context.TODO())

	if Config.HistoryOldMethod {
		if len(result) > 1 {
			_, errDel := collection.DeleteMany(context.TODO(), filter)
			logError("More than one in History!", currency.Symbol, 3)
			if errDel != nil {
				logError("DB problem!", errDel.Error(), 2)
				log.Fatalf("DB problem!")
			}
			AddHistory(currency)
			return
		}
		if len(result) == 1 {
			Updatehistory(currency)
			return
		}
		if len(result) < 1 {
			AddHistory(currency)
			return
		}
	}
}

// AddHistory - add new history record
func AddHistory(currency Quote) {
	start := time.Now()
	errPing := client.Ping(context.TODO(), nil)
	if errPing != nil {
		client = dbConnect()
		logError("AddHistory - DB connection is lost!", errPing.Error(), 3)
	}
	var h HistoryQuote
	h.Category = currency.Category
	var layout = "01-02-2006"
	var date = time.Now().Format("01-02-2006")
	h.Date, _ = time.Parse(layout, date)
	h.Rate = currency.Rate
	h.Symbol = currency.Symbol
	collection := client.Database("Quotes").Collection("History")
	_, err := collection.InsertOne(context.TODO(), h)
	if err != nil {
		logError("DB problem!", err.Error(), 2)
		log.Fatalf("DB problem!")
	}
	elapsed := int64(time.Since(start) / time.Millisecond)
	if Config.LogDebug {
		logEvent(7, "History writed", 200, currency.Symbol, elapsed)
	}
}

// Updatehistory - update existing history
func Updatehistory(currency Quote) {
	start := time.Now()
	var date = time.Now().Format("01-02-2006")
	var layout = "01-02-2006"
	var d, _ = time.Parse(layout, date)
	errPing := client.Ping(context.TODO(), nil)
	if errPing != nil {
		client = dbConnect()
		logError("Updatehistory - DB connection is lost!", errPing.Error(), 3)
	}
	collection := client.Database("Quotes").Collection("History")
	filter := bson.D{
		primitive.E{Key: "symbol", Value: currency.Symbol},
		primitive.E{Key: "category", Value: currency.Category},
		primitive.E{Key: "date", Value: d},
	}
	update := bson.D{
		primitive.E{Key: "$set", Value: bson.D{
			primitive.E{Key: "rate", Value: currency.Rate},
		}},
	}
	_, err := collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		logError("DB problem!", err.Error(), 2)
		log.Fatalf("DB problem!")
	}
	elapsed := int64(time.Since(start) / time.Millisecond)
	if Config.LogDebug {
		logEvent(7, "History updated", 200, currency.Symbol, elapsed)
	}
}

func loadHistory(s string, c int, t int) map[string]float64 {
	var r = make(map[string]float64)

	errPing := client.Ping(context.TODO(), nil)
	if errPing != nil {
		client = dbConnect()
		logError("loadHistory - DB connection is lost!", errPing.Error(), 3)
	}
	collection := client.Database("Quotes").Collection("History")
	filter := bson.D{
		primitive.E{Key: "symbol", Value: s},
		primitive.E{Key: "category", Value: c},
	}
	options := options.Find()
	options.SetLimit(int64(t))
	options.SetSort(bson.D{primitive.E{Key: "date", Value: -1}})
	cur, err := collection.Find(context.TODO(), filter, options)
	if err != nil {
		logError("DB problem!", err.Error(), 2)
		log.Fatalf("DB problem!")
	}

	for cur.Next(context.TODO()) {
		var elem HistoryQuote
		err := cur.Decode(&elem)
		if err != nil {
			logError("DB problem!", err.Error(), 2)
			log.Fatalf("DB problem!")
		}
		r[elem.Date.Format("2006-01-02")] = elem.Rate
	}
	if err := cur.Err(); err != nil {
		logError("DB problem!", err.Error(), 2)
		log.Fatalf("DB problem!")
	}

	cur.Close(context.TODO())
	return r
}

// Subscriptions

func isSubscriptionInDB(s Subscription) bool {
	var result []*Subscription
	errPing := client.Ping(context.TODO(), nil)
	if errPing != nil {
		client = dbConnect()
		logError("isSubscriptionInDB - DB connection is lost!", errPing.Error(), 3)
	}
	collection := client.Database("Quotes").Collection("Subscriptions")
	filter := bson.D{
		primitive.E{Key: "deviceid", Value: s.DeviceID},
		primitive.E{Key: "token", Value: s.Token},
		primitive.E{Key: "type", Value: s.Type},
		primitive.E{Key: "base", Value: s.Base},
		primitive.E{Key: "currency", Value: s.Currency},
		primitive.E{Key: "price", Value: s.Price},
		primitive.E{Key: "condition", Value: s.Condition},
	}
	cur, err := collection.Find(context.TODO(), filter)
	if err != nil {
		logError("DB problem!", err.Error(), 2)
		log.Fatalf("DB problem!")
	}
	for cur.Next(context.TODO()) {
		var elem Subscription
		err := cur.Decode(&elem)
		if err != nil {
			logError("DB problem!", err.Error(), 2)
			log.Fatalf("DB problem!")
		}
		result = append(result, &elem)
	}
	if err := cur.Err(); err != nil {
		logError("DB problem!", err.Error(), 2)
		log.Fatalf("DB problem!")
	}
	cur.Close(context.TODO())
	if len(result) > 1 {
		logError("Too many tokens", "Number of Subscriptions for "+s.Token+" is "+strconv.Itoa(len(result)), 4)
	}
	if len(result) >= 1 {
		return true
	}
	return false
}

func writeNewSubscription(s Subscription) {
	errPing := client.Ping(context.TODO(), nil)
	if errPing != nil {
		client = dbConnect()
		logError("writeNewSubscription - DB connection is lost!", errPing.Error(), 3)
	}
	collection := client.Database("Quotes").Collection("Subscriptions")
	_, err := collection.InsertOne(context.TODO(), s)
	if err != nil {
		logError("DB problem!", err.Error(), 2)
		log.Fatalf("DB problem!")
	}
}
