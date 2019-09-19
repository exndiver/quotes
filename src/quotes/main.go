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

// Locales - list of all Currencies titles
var Locales = loadLocales()

// Quote - Struct for qoute
type Quote struct {
	Symbol   string  `json:"symbol"`
	Rate     float64 `json:"rate"`
	Category int     `json:"category"`
}

// HistoryQuote - Struct for History
type HistoryQuote struct {
	Symbol   string    `json:"symbol"`
	Category int       `json:"category"`
	Date     time.Time `json:"date"`
	Rate     float64   `json:"rate"`
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

	if Config.Plugins.Amd {
		AMD()
	}

	if Config.Plugins.Gel {
		GEL()
	}

	if Config.Plugins.OpenExRates {
		openexchangerates()
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

	r.HandleFunc("/api/GetTitles/{locale}/", getTitles).Methods("GET")

	r.HandleFunc("/api/GetHistory/{d}/{c}/{s}", getHistoryMethod).Methods("GET")

	fmt.Printf("Starting server...\n")

	log.Print(http.ListenAndServe(Config.Hosts.Service, handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(r)))
}
