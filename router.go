package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/exndiver/feedback"
	"github.com/exndiver/feedback/googlesheet"

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
	if d < 1 {
		d = 1
	}
	if err != nil {
		w.Write([]byte("OK! Nothing!\n"))
	}
	c, err := strconv.Atoi(vars["c"])
	if err != nil {
		w.Write([]byte("OK! Nothing!\n"))
	}
	w.Write(getHistory(vars["s"], c, d))
}

func cachedHistory(duration string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Logger1(r)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		content := storage.Get(r.RequestURI)
		if content != nil {
			w.Write(content)
		} else {
			vars := mux.Vars(r)
			d, err := strconv.Atoi(vars["d"])
			if d < 1 {
				d = 1
			}
			if err != nil {
				w.Write([]byte("OK! Nothing!\n"))
			}
			c, err := strconv.Atoi(vars["c"])
			if err != nil {
				w.Write([]byte("OK! Nothing!\n"))
			}
			content = getHistory(vars["s"], c, d)
			if d, err := time.ParseDuration(duration); err == nil {
				storage.Set(r.RequestURI, content, d)
			} else {
				fmt.Printf("Page not cached. err: %s\n", err)
			}
			w.Write(content)
		}
	})
}

func postFeedback(w http.ResponseWriter, r *http.Request) {
	Logger1(r)

	var message feedback.Message
	if r.FormValue("message") != "" {
		message = googlesheet.NewFeedback("", r.FormValue("message"))
		message.Send("19Rk6QACMPc5W2Am1_odkY5Mb8nZLFkouok0BYgvO-B4")
	}
	w.Write([]byte("OK!\n"))
}
