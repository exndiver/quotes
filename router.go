package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/exndiver/feedback"
	"github.com/exndiver/feedback/googlesheet"

	//	"encoding/json"
	"github.com/gorilla/mux"
)

// DefaultPage - Very Default responce
func DefaultPage(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	loggerAccess(r)
	w.Write([]byte("OK! Nothing!\n"))
	elapsed := int64(time.Since(start) / time.Millisecond)
	go reqLogger(r, elapsed, "Default")
}

func avialibleCurrencies(w http.ResponseWriter, r *http.Request) {
	loggerAccess(r)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseAvialibleCurrencies())
}

// Method to get all Rates with EUR Base
// Example: /api/GetRates
func getRatesAPI(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	loggerAccess(r)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	resp := getRatesFromCache()
	w.Write(resp)
	elapsed := int64(time.Since(start) / time.Millisecond)
	go reqLogger(r, elapsed, "GetRates")
}

// Method to get Rates with Base Currency
// Example: /api/GetRates/0/USD
func getRatesBasedAPI(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	loggerAccess(r)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	groupID, err := strconv.Atoi(vars["groupID"])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("OK! Nothing!\n"))
		elapsed := int64(time.Since(start) / time.Millisecond)
		go reqRespLogger(r, elapsed, "GetRatesBased", "OK! Nothing!", http.StatusNotFound)
		return
	}
	w.Write(getRatesBasedFromCache(groupID, vars["symbol"]))
	elapsed := int64(time.Since(start) / time.Millisecond)
	go reqLogger(r, elapsed, "GetRatesBased")
}

// Method to get Titles from file config/titles.json
// Example: /api/GetTitles/ru
func getTitles(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	loggerAccess(r)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	vars := mux.Vars(r)
	w.Write(getLocale(vars["locale"]))
	elapsed := int64(time.Since(start) / time.Millisecond)
	go reqRespLogger(r, elapsed, "getTitles", vars["locale"], 200)
}

func getHistoryMethod(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	loggerAccess(r)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	vars := mux.Vars(r)
	d, err := strconv.Atoi(vars["d"])
	if d < 1 {
		d = 1
	}
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("OK! Nothing!\n"))
		elapsed := int64(time.Since(start) / time.Millisecond)
		go reqRespLogger(r, elapsed, "getHistory", vars["d"]+" - Param is wrong", http.StatusNotFound)
		return
	}
	c, err := strconv.Atoi(vars["c"])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("OK! Nothing!\n"))
		elapsed := int64(time.Since(start) / time.Millisecond)
		go reqRespLogger(r, elapsed, "getHistory", vars["c"]+" - Param is wrong", http.StatusNotFound)
		return
	}
	w.Write(getHistory(vars["s"], c, d))
	elapsed := int64(time.Since(start) / time.Millisecond)
	go reqRespLogger(r, elapsed, "getHistory", vars["s"], 200)
}

func cachedHistory(duration string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		loggerAccess(r)
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
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("OK! Nothing!\n"))
				elapsed := int64(time.Since(start) / time.Millisecond)
				go reqRespLogger(r, elapsed, "cachedHistory", vars["d"]+" - Param is wrong", http.StatusNotFound)
				return
			}
			c, err := strconv.Atoi(vars["c"])
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("OK! Nothing!\n"))
				elapsed := int64(time.Since(start) / time.Millisecond)
				go reqRespLogger(r, elapsed, "cachedHistory", vars["c"]+" - Param is wrong", http.StatusNotFound)
				return
			}
			content = getHistory(vars["s"], c, d)
			if d, err := time.ParseDuration(duration); err == nil {
				storage.Set(r.RequestURI, content, d)
			} else {
				fmt.Printf("Page not cached. err: %s\n", err)
			}
			w.Write(content)
			elapsed := int64(time.Since(start) / time.Millisecond)
			go reqRespLogger(r, elapsed, "cachedHistory", vars["s"], 200)
		}
	})
}

// Method to POST Feedback
// Example: /api/SendFeedback
func postFeedback(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	loggerAccess(r)

	if r.FormValue("message") != "" {
		go pf(strings.Join(r.Header["X-Forwarded-For"], ","), r.FormValue("message"))
	}
	w.Write([]byte("OK!\n"))
	elapsed := int64(time.Since(start) / time.Millisecond)
	go reqLogger(r, elapsed, "postFeedbackRequest")
}

// Post feedback
func pf(c string, msg string) {
	start := time.Now()
	var message feedback.Message
	message = googlesheet.NewFeedback(c, msg)
	message.Send(Config.Feedback)
	elapsed := int64(time.Since(start) / time.Millisecond)
	go emptyReqLogger(elapsed, "postFeedbackSend")
}

// Subcribe for push msg
func subscribe(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	loggerAccess(r)
	var s Subscription
	s.Lang = "en"
	r2 := r
	_ = json.NewDecoder(r.Body).Decode(&s)
	fmt.Printf("%s", s.Token)
	if (s.Token != "") && (ValidType(s.Type)) && (s.DeviceID != "") {
		w.Write([]byte("OK!\n"))
		putSubscription(s)
		elapsed := int64(time.Since(start) / time.Millisecond)
		go reqLogger(r, elapsed, "Subscribe")
		return
	}
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte("Error!\n"))
	elapsed := int64(time.Since(start) / time.Millisecond)
	go reqRespLogger(r2, elapsed, "Subscribe", "Error", http.StatusForbidden)
}

// Update device token for push msg
func updateSubscription(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	loggerAccess(r)

	var s Subscription
	_ = json.NewDecoder(r.Body).Decode(&s)
	if (s.Token != "") && (s.DeviceID != "") {
		w.Write([]byte("OK!\n"))
		updSub(s)
		elapsed := int64(time.Since(start) / time.Millisecond)
		go reqLogger(r, elapsed, "updateSubscription")
		return
	}
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte("Error!\n"))
	elapsed := int64(time.Since(start) / time.Millisecond)
	go reqRespLogger(r, elapsed, "updateSubscription", "Error", http.StatusForbidden)
}
