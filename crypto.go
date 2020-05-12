package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

// CrypResp - responce from crypto source
type CrypResp struct {
	Ticker Cryp `json:"ticker"`
}

// Cryp - ticker
type Cryp map[string]string

func getCrypto() {
	var C CrypResp
	for _, v := range Config.Cryptoapilist {
		start := time.Now()
		resp, err := http.Get(v)
		if err != nil {
			d := int64(time.Since(start) / time.Millisecond)
			logEvent(4, "loadCrypto", 500, "Error calling crypto api for: "+v+". Error: "+err.Error(), d)
			return
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			d := int64(time.Since(start) / time.Millisecond)
			logEvent(4, "loadCrypto", 500, "Error getting responce body"+v+". Error: "+err.Error(), d)
			return
		}
		if err := json.Unmarshal(body, &C); err != nil {
			d := int64(time.Since(start) / time.Millisecond)
			logEvent(4, "loadCrypto", 500, "Error parsing JSON for"+v+". Error: "+err.Error(), d)
			return
		}
		var s, e = strconv.ParseFloat(C.Ticker["price"], 64)
		if e != nil {
			d := int64(time.Since(start) / time.Millisecond)
			logEvent(4, "loadCrypto", 500, "Error parsing the price for"+v+". Error: "+err.Error(), d)
			return
		}
		var q = Quote{
			Symbol:   C.Ticker["base"],
			Rate:     (1 / s),
			Category: 1,
		}
		if isElementInDB(q) {
			updateRate(q)
		} else {
			writeNewCurrency(q)
		}
		d := int64(time.Since(start) / time.Millisecond)
		if Config.LogLoadRatesInfo {
			logEvent(6, "loadCrypto", 200, "Was loaded successfully"+v, d)
		}
	}
}
