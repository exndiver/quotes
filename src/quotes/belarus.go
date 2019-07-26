package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// QuotesResponse - Responce from api with quotes
type QuotesResponse struct {
	CurOfficialRate float64 `json:"Cur_OfficialRate"`
}

func blrdRub() {
	var quotes QuotesResponse
	resp, err := http.Get("http://www.nbrb.by/API/ExRates/Rates/292?Periodicity=0")

	if err != nil {
		Logger2Errors("Error importing from blrd bank")
	}

	Logger2("Belarus Rub was imported")

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	if err := json.Unmarshal(body, &quotes); err != nil {
		fmt.Printf("%+s\n", err)
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
