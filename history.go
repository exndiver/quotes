package main

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/bson"
)


func historyDBUpdate(){
	if historyCountInDb() > 0 {
	//	historyUpdateInDB()
	historyRemoveTodayFromDB()
	historyInsertInDB()
	} else {
		historyInsertInDB()
	}

}

func historyCountInDb() int64{
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

func historyInsertInDB(){
	start := time.Now()
	var d, _ = time.Parse("01-02-2006", time.Now().Format("01-02-2006"))
	var h []interface{}
	for _, cur := range QutesinMemory {
		var n HistoryQuote
		n.Symbol = cur.Symbol
		n.Category = cur.Category
		n.Rate = cur.Rate
		n.Date = d
		h = append(h,&n)
	}
	errPing := client.Ping(context.TODO(), nil)
	if errPing != nil {
		client = dbConnect()
		logError("historyRemoveTodayFromDB - DB connection is lost!", errPing.Error(), 3)
		return
	}
	collection := client.Database("Quotes").Collection("History")
	_, err := collection.InsertMany(context.TODO(), h)
	if err != nil {
		logError("DB problem!", err.Error(), 2)
	}
	elapsed := int64(time.Since(start) / time.Millisecond)
	if Config.LogDebug {
		logEvent(7, "History added", 200, "", elapsed)
	}
}

func historyUpdateInDB(){
	start := time.Now()
	var d, _ = time.Parse("01-02-2006", time.Now().Format("01-02-2006"))
	errPing := client.Ping(context.TODO(), nil)
	if errPing != nil {
		client = dbConnect()
		logError("historyRemoveTodayFromDB - DB connection is lost!", errPing.Error(), 3)
		return
	}
	collection := client.Database("Quotes").Collection("History")
	for _, cur := range QutesinMemory {
		filter := bson.D{
			primitive.E{Key: "symbol", Value: cur.Symbol},
			primitive.E{Key: "category", Value: cur.Category},
			primitive.E{Key: "date", Value: d},
		}
		update := bson.D{
			primitive.E{Key: "$set", Value: bson.D{
				primitive.E{Key: "rate", Value: cur.Rate},
			}},
		}
		_, err := collection.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			logError("DB problem!", err.Error(), 2)
			return
		}
	}
	elapsed := int64(time.Since(start) / time.Millisecond)
	if Config.LogDebug {
		logEvent(7, "History updated", 200, "", elapsed)
	}
}



func historyRemoveTodayFromDB(){
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
	if Config.LogDebug {
		logEvent(7, "History updated", 200, "", elapsed)
	}
}