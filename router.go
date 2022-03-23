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
	"github.com/exndiver/feedback/telegram"

	"github.com/gorilla/mux"
)

// DefaultPage - Default responce
func DefaultPage(w http.ResponseWriter, r *http.Request) (int, string, int, string, string) {
	mn := "Default"
	level := 6
	code := http.StatusOK
	resp := []byte("Hello!")
	rbody := ""
	w.WriteHeader(code)
	w.Write(resp)
	return code, mn, level, string(resp), rbody
}

func avialibleCurrencies(w http.ResponseWriter, r *http.Request) (int, string, int, string, string) {
	mn := "GetAvialibleCurrencies"
	level := 6
	code := http.StatusOK
	resp := responseAvialibleCurrencies()
	rbody := ""
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
	return code, mn, level, string(resp), rbody
}

// Method to get all Rates with EUR Base
// Example: /api/GetRates
func getRatesAPI(w http.ResponseWriter, r *http.Request) (int, string, int, string, string) {
	mn := "GetRates"
	level := 6
	code := http.StatusOK
	rbody := ""
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	resp := getRatesFromCache()
	w.Write(resp)
	return code, mn, level, string(resp), rbody
}

// Method to get Rates with Base Currency
// Example: /api/GetRates/0/USD
func getRatesBasedAPI(w http.ResponseWriter, r *http.Request) (int, string, int, string, string) {
	mn := "GetRatesBased"
	level := 6
	code := http.StatusOK
	rbody := ""
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	groupID, err := strconv.Atoi(vars["groupID"])
	if err != nil {
		code = http.StatusNotFound
		level = 4
		resp := []byte("Not Found")
		w.WriteHeader(code)
		w.Write(resp)
		return code, mn, level, string(resp), rbody
	}
	resp := getRatesBasedFromCache(groupID, vars["symbol"])
	w.Write(resp)
	return code, mn, level, string(resp), rbody
}

// Method to get Titles from file config/titles.json
// Example: /api/GetTitles/ru
func getTitles(w http.ResponseWriter, r *http.Request) (int, string, int, string, string) {
	mn := "GetTitles"
	level := 6
	code := http.StatusOK
	rbody := ""
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	vars := mux.Vars(r)
	resp := getLocale(vars["locale"])
	w.Write(resp)
	return code, mn, level, string(resp), rbody
}

//GetHistory method
// Ecample /api/GetHistory/30/0/RUB
func getHistoryCache(w http.ResponseWriter, r *http.Request) (int, string, int, string, string) {
	mn := "GetHistory"
	level := 6
	code := http.StatusOK
	rbody := ""
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	resp := storage.Get(r.RequestURI)
	if resp != nil {
		w.Write(resp)
		mn := "GetHistoryCached"
		return code, mn, level, string(resp), rbody
	}
	resp, status := getHistoryFromDB(r)
	if !status {
		code := http.StatusNotAcceptable
		w.WriteHeader(code)
		level = 4
	}
	w.Write(resp)
	mn = "GetHistoryDB"
	return code, mn, level, string(resp), rbody
}

func getHistoryFromDB(r *http.Request) ([]byte, bool) {
	vars := mux.Vars(r)
	d, err := strconv.Atoi(vars["d"])
	if d < 1 {
		d = 1
	}
	if err != nil {
		resp := []byte("Days param is wrong")
		return resp, false
	}
	c, err := strconv.Atoi(vars["c"])
	if err != nil {
		resp := []byte("Category param is wrong")
		return resp, false
	}
	resp := getHistory(vars["s"], c, d)
	if d, err := time.ParseDuration(Config.CacheDuration); err == nil {
		storage.Set(r.RequestURI, resp, d)
	} else {
		logEvent(4, "HistoryCache", 500, fmt.Sprintf("Page not cached. err: %s\n", err), 0)
	}
	return resp, true
}

// Method to POST Feedback
// Example: /api/SendFeedback
func postFeedback(w http.ResponseWriter, r *http.Request) (int, string, int, string, string) {
	mn := "PostFeedbackRequest"
	level := 6
	code := http.StatusOK
	resp := []byte("Sent!")
	rbody := r.FormValue("message")
	if r.FormValue("message") != "" {
		if Config.Feedback.Type == "googlesheet" {
			go pf(strings.Join(r.Header["X-Forwarded-For"], ","), rbody)
		} else {
			go pfT(strings.Join(r.Header["X-Forwarded-For"], ","), rbody)
		}

	}
	w.Write(resp)
	return code, mn, level, string(resp), rbody
}

// Post feedback
func pf(c string, msg string) {
	start := time.Now()
	var message feedback.Message = googlesheet.NewFeedback(c, msg)
	text, status := message.Send(Config.Feedback.Googlesheet)
	elapsed := int64(time.Since(start) / time.Millisecond)
	if !status {
		logEvent(3, "FeedbackSend", 500, text, elapsed)
	}
	logEvent(6, "FeedbackSend", 200, text, elapsed)
}

// Post feedback
func pfT(c string, msg string) {
	start := time.Now()
	var message feedback.Message = telegram.NewFeedback(c, msg, Config.Feedback.Telegram.ChatID)
	text, status := message.Send(Config.Feedback.Telegram.BotToken)
	elapsed := int64(time.Since(start) / time.Millisecond)
	if !status {
		logEvent(3, "FeedbackSend", 500, text, elapsed)
	}
	logEvent(6, "FeedbackSend", 200, text, elapsed)
}

// Subcribe for push msg
func subscribe(w http.ResponseWriter, r *http.Request) (int, string, int, string, string) {
	mn := "Subscribe"
	level := 6
	code := http.StatusOK
	resp := []byte("Done")
	var s Subscription
	s.Lang = "en"
	_ = json.NewDecoder(r.Body).Decode(&s)
	rbodyB, _ := json.Marshal(s)
	rbody := string(rbodyB)
	if (s.Token != "") && (ValidType(s.Type)) && (s.DeviceID != "") {
		w.Write(resp)
		putSubscription(s)
	} else {
		level = 4
		code = http.StatusForbidden
		resp = []byte("Forbidden")
		w.WriteHeader(code)
		w.Write(resp)
	}
	return code, mn, level, string(resp), rbody
}

// Update device token for push msg
func updateSubscription(w http.ResponseWriter, r *http.Request) (int, string, int, string, string) {
	mn := "updateSubscription"
	level := 6
	rbody := ""
	code := http.StatusOK
	resp := []byte("OK")
	var s Subscription
	_ = json.NewDecoder(r.Body).Decode(&s)
	if (s.Token != "") && (s.DeviceID != "") {
		w.Write(resp)
		updSub(s)
	} else {
		level = 4
		code = http.StatusForbidden
		resp = []byte("Forbidden")
		w.WriteHeader(code)
		w.Write(resp)
	}
	return code, mn, level, string(resp), rbody
}
