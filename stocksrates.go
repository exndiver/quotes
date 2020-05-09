package main

import (
	"io/ioutil"
	"net/http"
	"time"
)

func stockRate() {
	start := time.Now()

	for _, v := range Config.Stocks {
		resp, err := http.Get(v.Host)
		if err != nil {
			d := int64(time.Since(start) / time.Millisecond)
			logEvent(4, "loadStocks - "+v.Host, 500, "Error importing from stocks", d)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			d := int64(time.Since(start) / time.Millisecond)
			logEvent(4, "loadStocks - "+v.Host, 500, "Error parsing JSON stocks", d)
			return
		}
		q := 0.0
		status := true
		switch v.Name {
		case "EU":
			q, status = getEURates(v.Currency, body)
			if !status {
				d := int64(time.Since(start) / time.Millisecond)
				logEvent(4, "loadStocks - "+v.Host, 500, "Error parsing JSON stocks", d)
				continue
			}
		default:
			continue
		}

		str := Quote{
			Symbol:   v.Currency,
			Rate:     q,
			Category: 0,
		}

		if isElementInDB(str) {
			updateRate(str)
		} else {
			writeNewCurrency(str)
		}
		d := int64(time.Since(start) / time.Millisecond)
		if Config.LogLoadRatesInfo {
			logEvent(6, "loadOpenExchangerates", 200, "Was loaded successfully", d)
		}
	}
}
