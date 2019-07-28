package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

// QuotesUkr - Responce from api with quotes
type QuotesUkr struct {
	Rate float64 `json:"rate"`
}

// UkrUAH - loading UAH rates
func UkrUAH() {
	var quotes []QuotesUkr
	resp, err := http.Get("https://bank.gov.ua/NBUStatService/v1/statdirectory/exchange?valcode=EUR&json")

	if err != nil {
		Logger2Errors("Error importing from UKR bank")
	}

	Logger2("Ukr was imported")

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Logger2Errors("Error ioutil.ReadAll for UKR ")
		return
	}
	if err := json.Unmarshal(body, &quotes); err != nil {
		Logger2Errors("Error parsing JSON for UKR")
		return
	}
	str := Quote{
		Symbol:   "UAH",
		Rate:     quotes[0].Rate,
		Category: 0,
	}
	if isElementInDB(str) {
		updateRate(str)
	} else {
		writeNewCurrency(str)
	}
}
