package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

// QuotesResponse - struct for requesting all quotes from api exch
type QuotesResponse struct {
	Rates Quotes `json:"rates"`
}

// Quotes struct for each cur from api exch
type Quotes map[string]float64

func exchangeratesapi() {
	var quotes QuotesResponse
	resp, err := http.Get("https://api.exchangeratesapi.io/latest")

	if err != nil {
		Logger2Errors("Error importing from exchangeratesapi")
		return
	}

	Logger2("Apiexchange is imported")

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Logger2Errors("Error parsing JSON api.exchangeratesapi.io ")
		return
	}
	if err := json.Unmarshal(body, &quotes); err != nil {
		Logger2Errors("Error parsing JSON api.exchangeratesapi.io")
		return
	}
	for k, v := range quotes.Rates {
		str := Quote{
			Symbol:   k,
			Rate:     v,
			Category: 0,
		}
		if isElementInDB(str) {
			updateRate(str)
		} else {
			writeNewCurrency(str)
		}
	}
	str := Quote{
		Symbol:   "EUR",
		Rate:     1.0,
		Category: 0,
	}
	if isElementInDB(str) {
		updateRate(str)
	} else {
		writeNewCurrency(str)
	}
}