package main

import (
	"net/http"
	"fmt"
	"io/ioutil"
	"strconv"
	"encoding/json"
)

type CrypResp struct{
	Ticker	Cryp `json:"ticker"`
}

type Cryp map[string]string

func getCrypto(){
	var C CrypResp
	for _, v := range Config.Cryptoapilist{
		resp, err := http.Get(v)
		if err != nil {
			Logger2Errors("Error calling crypto api for %s" + v)
		}
		Logger2("%s was loaded successfully" + v)
		body,err := ioutil.ReadAll(resp.Body)
		if err != nil{
			Logger2Errors("Error getting responce body "+v)
		}
		if err := json.Unmarshal(body, &C); err != nil {
			Logger2Errors("Error parsing JSON for " + v)
			fmt.Printf("%v\n", err)
			return
		}
		var s, e = strconv.ParseFloat(C.Ticker["price"], 64)
		if e != nil {
			Logger2Errors("Error parsing the price for "+v)
		}
		var q = Quote {
			Symbol: C.Ticker["base"],
			Rate: 1/s,
			Category: 1,
		}
		if isElementInDB(q) {
			updateRate(q)
		} else {
			writeNewCurrency(q)
		}
	}
}