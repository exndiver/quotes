package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// OpenExResponse - response from openexachangerates api
type OpenExResponse struct {
	Rates Quotes `json:"rates"`
}

func openexchangerates() {
	start := time.Now()
	var quotes OpenExResponse
	resp, err := http.Get(Config.OpenExRateLink)
	if err != nil {
		d := int64(time.Since(start) / time.Millisecond)
		logEvent(4, "loadOpenExchangerates", 500, "Error importing from openexchangerates", d)
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		d := int64(time.Since(start) / time.Millisecond)
		logEvent(4, "loadOpenExchangerates", 500, "Error parsing JSON openexchangerates.org", d)
		return
	}
	if err := json.Unmarshal(body, &quotes); err != nil {
		d := int64(time.Since(start) / time.Millisecond)
		logEvent(4, "loadOpenExchangerates", 500, "Error parsing JSON openexchangerates.org", d)
		return
	}
	// Calculate USD rate. The api uses base currency USD, so to calculate other currencies rate to EUR: Rate(USD)*Rate(CurFromAPI)
	u := 1 / quotes.Rates["EUR"]

	// If USD is overided
	if Config.Stocks["USD"].Enable {
		for _, cur := range QutesinMemory {
			if cur.Category == 0 && cur.Symbol == "USD" {
				u = cur.Rate
				break
			} else {
				u = 1 / quotes.Rates["EUR"]
			}
		}
	}
	for _, v := range strings.Split(Config.OpenExRateCurList, ",") {
		if !Config.Stocks[v].Enable {
			var r = u * quotes.Rates[v]
			if v == "EUR" {
				r = 1
			}
			str := Quote{
				Symbol:   v,
				Rate:     r,
				Category: 0,
			}
			if v == "BYN" {
				str2 := Quote{
					Symbol:   "BYR",
					Rate:     r,
					Category: 0,
				}
				if isElementInDB(str2) {
					updateRate(str2)
				} else {
					writeNewCurrency(str2)
				}
			}
			if isElementInDB(str) {
				updateRate(str)
			} else {
				writeNewCurrency(str)
			}
		}
	}

	for _, v := range strings.Split(Config.OpenExRateMetalList, ",") {
		var r = u * quotes.Rates[v]
		str := Quote{
			Symbol:   v,
			Rate:     r,
			Category: 2,
		}
		if isElementInDB(str) {
			updateRate(str)
		} else {
			writeNewCurrency(str)
		}
	}
	d := int64(time.Since(start) / time.Millisecond)
	if Config.LogLoadRatesInfo {
		logEvent(6, "loadOpenExchangerates", 200, "Was loaded successfully", d)
	}
}
