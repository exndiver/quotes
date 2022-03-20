package main

import (
	"encoding/json"
	"math"
	"sort"
	"strings"
)

//Response - is just a responce
type Response struct {
	Type  string
	Value []string
}

func getRatesFromCache() []byte {
	var response = make(map[string](map[string]float64))
	for index, category := range strings.Split(Config.AvialibleTypes, ",") {
		var currency = make(map[string]float64)
		for _, cur := range QutesinMemory {
			if cur.Category == index {
				currency[cur.Symbol] = cur.Rate
			}
		}
		response[category] = currency
	}
	jsonResult, _ := json.Marshal(response)
	return jsonResult
}

func getRatesBasedFromCache(groupID int, symbol string) []byte {
	var response = make(map[string](map[string]float64))

	var delta = 1.0

	for _, cur := range QutesinMemory {
		if cur.Category == groupID {
			if cur.Symbol == strings.ToUpper(symbol) {
				delta = 1 / cur.Rate
				break
			}
		}
	}
	for index, category := range strings.Split(Config.AvialibleTypes, ",") {
		var currency = make(map[string]float64)
		for _, cur := range QutesinMemory {
			if cur.Category == index {
				var newRate = math.Round((cur.Rate*delta)*10000000000) / 10000000000
				if cur.Symbol == strings.ToUpper(symbol) {
					newRate = 1.0
				}
				currency[cur.Symbol] = newRate
			}
		}
		response[category] = currency
	}
	jsonResult, _ := json.Marshal(response)
	return jsonResult
}

func responseAvialibleCurrencies() []byte {
	var r []Response

	for _, Type := range strings.Split(Config.AvialibleTypes, ",") {
		var temArr []string
		temArr = append(temArr, strings.Split(Config.AvialibleList[Type], ",")...)
		sort.Strings(temArr)
		r = append(r, Response{Type, temArr})
	}
	jsonResult, _ := json.Marshal(r)
	return jsonResult
}

func getHistory(symbol string, groupID int, d int) []byte {
	jsonResult, _ := json.Marshal(loadHistory(symbol, groupID, d))
	return jsonResult
}
