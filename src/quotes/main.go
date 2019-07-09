package main

import (
	"fmt"
	"log"
	"net/http"
	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
	"time"
)

var Config = get_config()

type Quote struct{
	Symbol string `json:"symbol"`
	Rate float64 `json:"rate"`
	Category int `json:"category"`
}

var QutesinMemory []* Quote
var LastQuotesUpdate = time.Now()
var LastQuotesReload = time.Now()

func reloadCurrenciesInMemory(){
	getAllElementsinMemory()
	LastQuotesReload = time.Now()
	nextTime := time.Now().Truncate(time.Hour)
	nextTime = nextTime.Add(time.Hour)
	time.Sleep(time.Until(nextTime))
	go reloadCurrenciesInMemory()
}

func updateQuotesInDB(){
	if Config.Plugins.Exchangeratesapi {
		exchangeratesapi()
	}
	LastQuotesUpdate = time.Now()
	nextTime := time.Now().Truncate(time.Hour * 12)
	nextTime = nextTime.Add(time.Hour * 12)
	time.Sleep(time.Until(nextTime))
	go updateQuotesInDB()
}

func main() {
	
	go reloadCurrenciesInMemory()

	go updateQuotesInDB()

	r := mux.NewRouter().StrictSlash(true)

	r.HandleFunc("/", DefaultPage).Methods("GET")

	r.HandleFunc("/api/GetAvialibleCurrencies/", avialibleCurrencies).Methods("GET")
	
	r.HandleFunc("/api/GetRates/", getRatesAPI).Methods("GET")

	r.HandleFunc("/api/GetRates/{groupID}/{symbol}", getRatesBasedAPI).Methods("GET")

	fmt.Printf("Starting server for testing HTTP POST...\n")
	log.Print(http.ListenAndServe(Config.Hosts.Service, handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(r)))
}	