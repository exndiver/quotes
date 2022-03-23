package main

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func historyDBUpdate() {
	if historyCountInDb() > 0 {
		historyRemoveTodayFromDB()
		historyInsertInDB()
		historyInsert("h")
	} else {
		historyInsertInDB()
		historyInsert("h")
	}
}

func historyStructToDB() []interface{} {
	var day = int(time.Now().Weekday())
	var d, _ = time.Parse("01-02-2006", time.Now().Format("01-02-2006"))
	var h []interface{}
	for _, cur := range QutesinMemory {
		var n HistoryQuote
		n.Symbol = cur.Symbol
		n.Category = cur.Category
		n.Rate = cur.Rate
		n.Date = d
		if cur.Category != 1 {
			if (day == 0) || (day == 6) {
				continue
			}
		}
		h = append(h, &n)
	}
	return h
}

func historyCountInDb() int64 {
	var d, _ = time.Parse("01-02-2006", time.Now().Format("01-02-2006"))
	errPing := client.Ping(context.TODO(), nil)
	if errPing != nil {
		client = dbConnect()
		logError("historyCheckInDb - DB connection is lost!", errPing.Error(), 3)
	}
	collection := client.Database("Quotes").Collection("History")
	filter := bson.D{
		primitive.E{Key: "date", Value: d},
	}
	itemCount, _ := collection.CountDocuments(context.TODO(), filter)
	return itemCount
}

func historyInsertInDB() {
	start := time.Now()
	h := historyStructToDB()
	errPing := client.Ping(context.TODO(), nil)
	if errPing != nil {
		client = dbConnect()
		logError("historyInsertInDB - DB connection is lost!", errPing.Error(), 3)
		return
	}
	collection := client.Database("Quotes").Collection("History")
	_, err := collection.InsertMany(context.TODO(), h)
	if err != nil {
		logError("DB problem!", err.Error(), 2)
	}
	elapsed := int64(time.Since(start) / time.Millisecond)
	logEvent(7, "History added", 200, "", elapsed)
}

func getHistoryStruct(p string) []interface{} {
	var f string
	switch p {
	case "d":
		f = "01-02-2006"
	case "h":
		f = "01-02-2006 15:02:00"
	default:
		f = "01-02-2006"
	}
	var day = int(time.Now().Weekday())
	var d, _ = time.Parse(f, time.Now().Format(f))
	var h []interface{}
	for _, cur := range QutesinMemory {
		var n HistoryQuote
		n.Symbol = cur.Symbol
		n.Category = cur.Category
		n.Rate = cur.Rate
		n.Date = d
		if cur.Category != 1 {
			if (day == 0) || (day == 6) {
				continue
			}
		}
		h = append(h, &n)
	}
	return h
}

// Insert or update history record; p - period (d - day, h  hour)
func historyInsert(p string) {
	// Collection name based on period p
	var c string
	start := time.Now()
	h := getHistoryStruct(p)
	errPing := client.Ping(context.TODO(), nil)
	if errPing != nil {
		client = dbConnect()
		logError("historyInsert - DB connection is lost! : "+p, errPing.Error(), 3)
		return
	}
	removeHistoryDate(p)
	switch p {
	case "d":
		c = "History"
	case "h":
		c = "History_1h"
	default:
		c = "History"
	}
	collection := client.Database("Quotes").Collection(c)
	_, err := collection.InsertMany(context.TODO(), h)
	if err != nil {
		logError("DB problem! : "+p, err.Error(), 2)
	}
	elapsed := int64(time.Since(start) / time.Millisecond)
	logEvent(7, "historyInsert", 200, "History inserted "+p, elapsed)
}

func removeHistoryDate(p string) {
	start := time.Now()
	var c string
	var f string
	switch p {
	case "d":
		c = "History"
		f = "01-02-2006"
	case "h":
		c = "History_1h"
		f = "01-02-2006 15:02:00"
	default:
		c = "History"
		f = "01-02-2006"
	}
	var d, _ = time.Parse(f, time.Now().Format(f))
	errPing := client.Ping(context.TODO(), nil)
	if errPing != nil {
		client = dbConnect()
		logError("removeHistoryDate - DB connection is lost!", errPing.Error(), 3)
	}
	collection := client.Database("Quotes").Collection(c)
	filter := bson.D{
		primitive.E{Key: "date", Value: d},
	}
	_, errDel := collection.DeleteMany(context.TODO(), filter)
	if errDel != nil {
		logError("DB problem!", errDel.Error(), 2)
		return
	}

	elapsed := int64(time.Since(start) / time.Millisecond)
	logEvent(7, "removeHistoryDate", 200, "History removed "+c+" "+f, elapsed)

}

func historyRemoveTodayFromDB() {
	start := time.Now()
	var d, _ = time.Parse("01-02-2006", time.Now().Format("01-02-2006"))
	errPing := client.Ping(context.TODO(), nil)
	if errPing != nil {
		client = dbConnect()
		logError("historyRemoveTodayFromDB - DB connection is lost!", errPing.Error(), 3)
	}
	collection := client.Database("Quotes").Collection("History")
	filter := bson.D{
		primitive.E{Key: "date", Value: d},
	}
	_, errDel := collection.DeleteMany(context.TODO(), filter)
	if errDel != nil {
		logError("DB problem!", errDel.Error(), 2)
		return
	}

	elapsed := int64(time.Since(start) / time.Millisecond)
	logEvent(7, "History updated", 200, "", elapsed)
}
