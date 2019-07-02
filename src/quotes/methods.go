package main

import (
	"strings"
	//"fmt"
	"encoding/json"
)

type Response struct{
	Type string
	Value []string
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