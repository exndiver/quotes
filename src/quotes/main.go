package main

import (
	"fmt"
	"log"
	"net/http"
	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
)
var Config = get_config()

type Quote struct{
	Symbol string `json:"symbol"`
	Rate float64 `json:"rate"`
	Category int `json:"category"`
}

func main() {
	r := mux.NewRouter().StrictSlash(true)

	r.HandleFunc("/", DefaultPage).Methods("GET")

	r.HandleFunc("/api/GetAvialibleCurrencies/", avialibleCurrencies).Methods("GET")
	
	r.HandleFunc("/api/GetRates/", getRatesAPI).Methods("GET")

	r.HandleFunc("/api/GetRates/{groupID}/{symbol}", getRatesBasedAPI).Methods("GET")

	fmt.Printf("Starting server for testing HTTP POST...\n")
	log.Print(http.ListenAndServe(Config.Hosts.Service, handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(r)))
}	