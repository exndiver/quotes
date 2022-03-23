package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/exndiver/cache"
	"github.com/exndiver/cache/memory"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// storage - cache from pkg github.com/exndiver/cache
var storage cache.Storage

// Config - main configuration from config.json file
var Config = getConfig()

// Locales - list of all Currencies titles
var Locales = loadLocales()

var client = dbConnect()

// Quote - Struct for qoute
type Quote struct {
	Symbol   string  `json:"symbol"`
	Rate     float64 `json:"rate"`
	Category int     `json:"category"`
}

// Quotes struct for each cur from api exch
type Quotes map[string]float64

// HistoryQuote - Struct for History
type HistoryQuote struct {
	Symbol   string    `json:"symbol"`
	Category int       `json:"category"`
	Date     time.Time `json:"date"`
	Rate     float64   `json:"rate"`
}

// QutesinMemory - in memory cache of all quotes in db
var QutesinMemory []*Quote

// currencyHourTimer - Currency Updater
func currencyHourTimer() {
	var d = time.Duration(1)
	var day = int(time.Now().Weekday())
	nextTime := time.Now().Truncate(time.Hour * d)
	nextTime = nextTime.Add(time.Hour * d)

	if (day == 0) || (day == 6) {
		logEvent(7, "CurrenciesUpdate", 400, "Skip update. It is weekend: day "+strconv.Itoa(day), 0)
	} else {
		// Check plugins and Update

		if Config.Plugins.OpenExRates {
			openexchangerates()
		}
		logEvent(7, "CurrenciesUpdate", 400, "Currencies updated "+strconv.Itoa(day), 0)
	}
	time.Sleep(time.Until(nextTime))
	go currencyHourTimer()
}

func reloadCurrenciesInMemoryAsync() {
	getAllElementsinMemory()
	nextTime := time.Now().Truncate(time.Minute * 5)
	nextTime = nextTime.Add(time.Minute * 5)
	historyDBUpdate()
	time.Sleep(time.Until(nextTime))
	go reloadCurrenciesInMemoryAsync()
}

func updateQuotesCryptocurrenciesInDB() {
	if Config.Plugins.Crypto {
		getCrypto()
	}

	nextTime := time.Now().Truncate(time.Minute * 1)
	nextTime = nextTime.Add(time.Minute * 1)
	time.Sleep(time.Until(nextTime))
	go updateQuotesCryptocurrenciesInDB()
}

func updateStocks() {
	stockRate()
	nextTime := time.Now().Truncate(time.Minute * 5)
	nextTime = nextTime.Add(time.Minute * 5)
	time.Sleep(time.Until(nextTime))
	go updateStocks()
}

func serverPrep() {
	start := time.Now()
	fmt.Printf("Load all quotes from db\n")
	getAllElementsinMemory()
	if Config.DownloadRates {
		stockRate()
		if Config.Plugins.OpenExRates {
			openexchangerates()
		}
		if Config.Plugins.Crypto {
			getCrypto()
		}
	}
	d := int64(time.Since(start) / time.Millisecond)
	fmt.Printf("Server was prepared in %dms\n", d)
}

func main() {

	storage = memory.NewStorage()
	serverPrep()

	if Config.DownloadRates {
		fmt.Printf("Downloading quotes..\n")
		go currencyHourTimer()
		go updateStocks()
		go updateQuotesCryptocurrenciesInDB()
	} else {
		fmt.Printf("Downloading is swithced off..\n")
	}

	go reloadCurrenciesInMemoryAsync()

	r := mux.NewRouter().StrictSlash(true)

	r.Handle("/", logger(DefaultPage)).Methods("GET")

	r.Handle("/api/GetAvialibleCurrencies/", logger(avialibleCurrencies)).Methods("GET")

	r.Handle("/api/GetRates/", logger(getRatesAPI)).Methods("GET")

	r.Handle("/api/GetRates/{groupID}/{symbol}", logger(getRatesBasedAPI)).Methods("GET")

	r.Handle("/api/GetTitles/{locale}/", logger(getTitles)).Methods("GET")

	r.Handle("/api/GetHistory/{d}/{c}/{s}", logger(getHistoryCache)).Methods("GET")

	r.Handle("/api/SendFeedback", logger(postFeedback)).Methods("POST")

	r.Handle("/api/Subscribe", logger(subscribe)).Methods("POST")

	r.Handle("/api/UpdateSubscription", logger(updateSubscription)).Methods("POST")

	fmt.Printf("Starting server...\n")

	log.Print(http.ListenAndServe(Config.Hosts.Service, handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(r)))
}

func logger(endpoint func(http.ResponseWriter, *http.Request) (int, string, int, string, string)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		code, mn, level, resp, reqBod := endpoint(w, r)

		elapsed := int64(time.Since(start) / time.Millisecond)

		var l jsonLog
		l.Duration = elapsed
		l.Level = level
		l.Method = mn
		l.RequestURI = r.RequestURI

		ip := r.RemoteAddr
		if r.Header["X-Forwarded-For"] != nil {
			ip = r.Header["X-Forwarded-For"][0]
		}
		//l.RequestRemoteAddress = r.RemoteAddr
		l.RequestRemoteAddress = ip
		l.Request = string(reqBod)
		reqH, _ := json.Marshal(r.Header)
		l.RequestHeaders = string(reqH)
		l.Response = resp
		l.ResponseCode = code

		loggerJSON(l)
	})
}
