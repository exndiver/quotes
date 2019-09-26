package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
)

// OpenExResponse - response from openexachangerates api
type OpenExResponse struct {
	Rates Quotes `json:"rates"`
}

func openexchangerates() {
	var quotes OpenExResponse
	resp, err := http.Get(Config.OpenExRateLink)
	if err != nil {
		Logger2Errors("Error importing from openexchangerates")
		return
	}

	Logger2("Apiexchange is imported")

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Logger2Errors("Error parsing JSON openexchangerates.org")
		return
	}
	if err := json.Unmarshal(body, &quotes); err != nil {
		Logger2Errors("Error parsing JSON openexchangerates.org")
		return
	}
	var u = 1 / quotes.Rates["EUR"]
	for _, v := range strings.Split(Config.OpenExRateCurList, ",") {
		if v == "EUR" {
			var r = 1
		} else {
			var r = u * quotes.Rates[v]
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
}
