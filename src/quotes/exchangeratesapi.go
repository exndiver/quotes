package main

import (
	"net/http"
	"fmt"
	"log"
	"io/ioutil"
	"encoding/json"
)

type QuotesResponse struct{
	Rates	Quotes `json:"rates"`
}

type Quotes map[string]float64

func exchangeratesapi(){
	var quotes QuotesResponse
	resp, err := http.Get("https://api.exchangeratesapi.io/latest")

	if err != nil{
		Logger2Errors("Error importing from exchangeratesapi")
	}

	Logger2("Apiexchange is imported")

	defer resp.Body.Close()

	body,err := ioutil.ReadAll(resp.Body)
	if err != nil{
		log.Fatalln(err)
	}
	if err := json.Unmarshal(body, &quotes); err != nil {
		fmt.Printf("%+s\n",err)
		return
	}
	for k,v := range quotes.Rates{
		str := Quote{
			Symbol: k,
			Rate: v,
			Category: 0,
		}
		if isElementInDB(str) {
			updateRate(str)
		} else {
			writeNewCurrency(str)
		}
	}
	str := Quote{
		Symbol: "EUR",
		Rate: 1.0,
		Category: 0,
	}
	if isElementInDB(str) {
		updateRate(str)
	} else {
		writeNewCurrency(str)
	}
}