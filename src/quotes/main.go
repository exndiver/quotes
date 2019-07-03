package main

import (
	"fmt"
	"log"
	"net/http"
	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
)
var Config = get_config()

func main() {
	r := mux.NewRouter().StrictSlash(true)

	r.HandleFunc("/", DefaultPage).Methods("GET")

	r.HandleFunc("/api/GetAvialibleCurrencies/", avialibleCurrencies).Methods("GET")


	fmt.Printf("Starting server for testing HTTP POST...\n")
	log.Print(http.ListenAndServe(Config.Hosts.Service, handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(r)))
}	