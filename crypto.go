package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
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
		resp, err := http.Get(v)
		if err != nil {
			loggerAPIErrors("Error calling crypto api for %s" + v)
			return
		}
		loggerAPI("Was loaded successfully " + v)
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			loggerAPIErrors("Error getting responce body " + v)
			return
		}
		if err := json.Unmarshal(body, &C); err != nil {
			loggerAPIErrors("Error parsing JSON for " + v)
			return
		}
		var s, e = strconv.ParseFloat(C.Ticker["price"], 64)
		if e != nil {
			loggerAPIErrors("Error parsing the price for " + v)
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
	}
}
