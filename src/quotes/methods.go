package main

import (
	"strings"
	"encoding/json"
	"math"
)

type Response struct{
	Type string
	Value []string
}


func getRatesFromCache()[]byte{
	var response = make (map[string](map[string]float64))
	for index, category := range strings.Split(Config.AvialibleTypes, ",") {
		var currency = make (map[string]float64)
		for _, cur := range QutesinMemory{
			if cur.Category == index{
				currency[cur.Symbol] = cur.Rate
			}
		}
		response[category] = currency
	}
	json_result, _ := json.Marshal(response)
	return json_result
}

func getRatesBasedFromCache(groupID int, symbol string)[]byte{
	var response = make (map[string](map[string]float64))

	var delta = 1.0

	for _, cur := range QutesinMemory{
		if cur.Category == groupID{
			if cur.Symbol == strings.ToUpper(symbol) {
				delta = 1/cur.Rate
				break
			}
		}
	}
	for index, category := range strings.Split(Config.AvialibleTypes, ",") {
		var currency = make (map[string]float64)
		for _, cur := range QutesinMemory{
			if cur.Category == index{
				var newRate = math.Round((cur.Rate*delta)*10000000000)/10000000000
				if cur.Symbol == strings.ToUpper(symbol) {
					newRate= 1.0
				}
				currency[cur.Symbol] = newRate
			}
		}
		response[category] = currency
	}
	json_result, _ := json.Marshal(response)
	return json_result
}

func responseAvialibleCurrencies()[]byte{
	var r []Response
	
	for _, Type := range strings.Split(Config.AvialibleTypes, ","){
		var temArr []string
		if Type == "Currencies" {
			for _, Cur := range strings.Split(Config.AvialibleList.Currencies,","){
				temArr = append(temArr, Cur)
			}
		}
		r = append(r, Response{Type,temArr}) 
	}
	json_result, _ := json.Marshal(r)
	return json_result
}