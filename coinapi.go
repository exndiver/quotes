package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// CrypResp - responce from crypto source
type CrypResp struct {
	Ticker Cryp `json:"ticker"`
}

type CoinApiResp struct {
	Rates []struct {
		Time         time.Time `json:"time"`
		AssetIDQuote string    `json:"asset_id_quote"`
		Rate         float64   `json:"rate"`
	} `json:"rates"`
}

// Cryp - ticker
type Cryp map[string]string

// func getCrypto() {
func getCrypto() {
	var C CoinApiResp

	start := time.Now()
	client := &http.Client{}
	req, _ := http.NewRequest("GET", Config.Coinapi["url"], nil)
	req.Header.Add("Accept", "text/plain")
	req.Header.Add("X-CoinAPI-Key", Config.Coinapi["key"])
	res, err := client.Do(req)
	if err != nil {
		d := int64(time.Since(start) / time.Millisecond)
		logEvent(4, "loadCrypto", 500, "Error parsing JSON openexchangerates.org: "+err.Error(), d)
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		d := int64(time.Since(start) / time.Millisecond)
		logEvent(4, "loadCrypto", 500, "Error parsing JSON openexchangerates.org: "+err.Error(), d)
		return
	}

	if err := json.Unmarshal(body, &C); err != nil {
		d := int64(time.Since(start) / time.Millisecond)
		logEvent(4, "loadCrypto", 500, "Error parsing JSON openexchangerates.org: "+err.Error(), d)
		return
	}

	for _, v := range strings.Split(Config.Coinapi["list"], ",") {
		for _, crypto := range C.Rates {
			if crypto.AssetIDQuote == v {
				fmt.Print(crypto.AssetIDQuote)
				fmt.Printf(" %.8f\n", crypto.Rate)

				var q = Quote{
					Symbol:   crypto.AssetIDQuote,
					Rate:     crypto.Rate,
					Category: 1,
				}

				if isElementInDB(q) {
					updateRate(q)
				} else {
					writeNewCurrency(q)
				}
			}
		}
	}
	d := int64(time.Since(start) / time.Millisecond)
	if Config.LogLoadRatesInfo {
		logEvent(6, "loadCrypto", 200, "Crypto Was loaded successfully ", d)
	}

}
