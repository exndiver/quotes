package main

import (
	"net/http"
	"strconv"

	//	"encoding/json"
	"github.com/gorilla/mux"
)

// DefaultPage - Very Default responce
func DefaultPage(w http.ResponseWriter, r *http.Request) {
	Logger1(r)
	w.Write([]byte("OK! Nothing!\n"))
}

func avialibleCurrencies(w http.ResponseWriter, r *http.Request) {
	Logger1(r)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseAvialibleCurrencies())
}

func getRatesAPI(w http.ResponseWriter, r *http.Request) {
	Logger1(r)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Write(getRatesFromCache())
}

func getRatesBasedAPI(w http.ResponseWriter, r *http.Request) {
	Logger1(r)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	groupID, err := strconv.Atoi(vars["groupID"])
	if err != nil {
		w.Write([]byte("OK! Nothing!\n"))
	}
	w.Write(getRatesBasedFromCache(groupID, vars["symbol"]))
}

func getTitles(w http.ResponseWriter, r *http.Request) {
	Logger1(r)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	vars := mux.Vars(r)
	w.Write(getLocale(vars["locale"]))
}

func getHistoryMethod(w http.ResponseWriter, r *http.Request) {
	Logger1(r)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	vars := mux.Vars(r)
	d, err := strconv.Atoi(vars["d"])
	if err != nil {
		w.Write([]byte("OK! Nothing!\n"))
	}
	c, err := strconv.Atoi(vars["c"])
	if err != nil {
		w.Write([]byte("OK! Nothing!\n"))
	}
	w.Write(getHistory(vars["s"], c, d))
}
