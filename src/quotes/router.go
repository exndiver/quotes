package main

import (
//	"fmt"
//	"io/ioutil"
	"net/http"
//	"encoding/json"
//	"github.com/gorilla/mux"
)

func DefaultPage(w http.ResponseWriter, r *http.Request) {
//	Logger1(r)
	w.Write([]byte("OK! Nothing!\n"))
}

func avialibleCurrencies(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseAvialibleCurrencies())
}