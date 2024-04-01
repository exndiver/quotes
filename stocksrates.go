package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func stockRate() {
	start := time.Now()
	for _, v := range Config.Stocks {
		if v.Enable {
			fmt.Println("stockRate")
			fmt.Println(v.Enable)
			resp, err := http.Get(v.Host)
			if err != nil {
				d := int64(time.Since(start) / time.Millisecond)
				logEvent(4, "loadStocks", 500, "Error importing from stocks "+v.Host+": "+err.Error(), d)
				return
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				d := int64(time.Since(start) / time.Millisecond)
				logEvent(4, "loadStocks", 500, "Error parsing JSON stocks "+v.Host+": "+err.Error(), d)
				return
			}
			q := 0.0
			status := true
			switch v.Name {
			case "EU":
				if v.Enable {
					q, status = getEURates(v.Currency, body)
					if !status {
						d := int64(time.Since(start) / time.Millisecond)
						logEvent(4, "loadStocks", 500, "Error parsing JSON stocks "+v.Host+": ", d)
						continue
					}
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
				logEvent(6, "loadStocks", 200, v.Host+" Was loaded successfully", d)
			}
		}
	}
}
