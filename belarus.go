package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

// QuotesBlrd - Responce from api with quotes
type QuotesBlrd struct {
	CurOfficialRate float64 `json:"Cur_OfficialRate"`
}

func blrdRub() {
	var quotes QuotesBlrd
	resp, err := http.Get("http://www.nbrb.by/API/ExRates/Rates/292?Periodicity=0")

	if err != nil {
		Logger2Errors("Error importing from blrd bank")
		return
	}

	Logger2("Belarus Rub was imported")

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Logger2Errors("Error ioutil.ReadAll for BLRD ")
		return
	}
	if err := json.Unmarshal(body, &quotes); err != nil {
		Logger2Errors("Error parsing JSON for BLRD")
		return
	}
	str := Quote{
		Symbol:   "BYR",
		Rate:     quotes.CurOfficialRate,
		Category: 0,
	}
	if isElementInDB(str) {
		updateRate(str)
	} else {
		writeNewCurrency(str)
	}
}
