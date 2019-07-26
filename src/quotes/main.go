package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// Config - main configuration from config.json file
var Config = get_config()

// Quote - Struct for qoute
type Quote struct {
	Symbol   string  `json:"symbol"`
	Rate     float64 `json:"rate"`
	Category int     `json:"category"`
}

// QutesinMemory - in memory cache of all quotes in db
var QutesinMemory []*Quote

func reloadCurrenciesInMemory() {
	getAllElementsinMemory()
	nextTime := time.Now().Truncate(time.Hour)
	nextTime = nextTime.Add(time.Hour)
	time.Sleep(time.Until(nextTime))
	go reloadCurrenciesInMemory()
}

func updateQuotesExchangeratesapiInDB() {
	if Config.Plugins.Exchangeratesapi {
		exchangeratesapi()
	}
	nextTime := time.Now().Truncate(time.Hour * 12)
	nextTime = nextTime.Add(time.Hour * 12)
	time.Sleep(time.Until(nextTime))
	go updateQuotesExchangeratesapiInDB()
}

func updateQuotesCryptocurrenciesInDB() {
	if Config.Plugins.Crypto {
		getCrypto()
	}
	nextTime := time.Now().Truncate(time.Minute * 10)
	nextTime = nextTime.Add(time.Minute * 10)
	time.Sleep(time.Until(nextTime))
	go updateQuotesCryptocurrenciesInDB()
}

func main() {

	go reloadCurrenciesInMemory()

	go updateQuotesExchangeratesapiInDB()

	go updateQuotesCryptocurrenciesInDB()

	r := mux.NewRouter().StrictSlash(true)

	r.HandleFunc("/", DefaultPage).Methods("GET")

	r.HandleFunc("/api/GetAvialibleCurrencies/", avialibleCurrencies).Methods("GET")

	r.HandleFunc("/api/GetRates/", getRatesAPI).Methods("GET")

	r.HandleFunc("/api/GetRates/{groupID}/{symbol}", getRatesBasedAPI).Methods("GET")

	fmt.Printf("Starting server for testing HTTP POST...\n")
	log.Print(http.ListenAndServe(Config.Hosts.Service, handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(r)))
}
