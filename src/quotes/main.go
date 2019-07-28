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
var Config = getConfig()

// Quote - Struct for qoute
type Quote struct {
	Symbol   string  `json:"symbol"`
	Rate     float64 `json:"rate"`
	Category int     `json:"category"`
}

// QutesinMemory - in memory cache of all quotes in db
var QutesinMemory []*Quote

// currencyTimer - Currency Updater
func currencyTimer() {
	nextTime := time.Now().Truncate(time.Hour * 2)
	nextTime = nextTime.Add(time.Hour * 2)
	// Check plugins and Update
	if Config.Plugins.Exchangeratesapi {
		exchangeratesapi()
	}

	if Config.Plugins.Blrd {
		blrdRub()
	}

	if Config.Plugins.Srb {
		SrbDinar()
	}

	if Config.Plugins.Ukr {
		UkrUAH()
	}

	if Config.Plugins.Kzt {
		KZT()
	}

	if Config.Plugins.Azt {
		AZT()
	}

	time.Sleep(time.Until(nextTime))

	go currencyTimer()
}

func reloadCurrenciesInMemory() {
	getAllElementsinMemory()
	nextTime := time.Now().Truncate(time.Minute * 5)
	nextTime = nextTime.Add(time.Minute * 5)
	time.Sleep(time.Until(nextTime))
	go reloadCurrenciesInMemory()
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

	go currencyTimer()

	go updateQuotesCryptocurrenciesInDB()

	r := mux.NewRouter().StrictSlash(true)

	r.HandleFunc("/", DefaultPage).Methods("GET")

	r.HandleFunc("/api/GetAvialibleCurrencies/", avialibleCurrencies).Methods("GET")

	r.HandleFunc("/api/GetRates/", getRatesAPI).Methods("GET")

	r.HandleFunc("/api/GetRates/{groupID}/{symbol}", getRatesBasedAPI).Methods("GET")

	fmt.Printf("Starting server for testing HTTP POST...\n")
	log.Print(http.ListenAndServe(Config.Hosts.Service, handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(r)))
}
