package main

import (
	"encoding/json"
)

// eu - response from openexachangerates api
type eu struct {
	Rates Quotes `json:"rates"`
}

func getEURates(c string, body []byte) (float64, bool) {
	var quotes eu
	if err := json.Unmarshal(body, &quotes); err != nil {
		return 0, false
	}
	return quotes.Rates[c], true
}
