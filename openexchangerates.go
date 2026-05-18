package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenExResponse - response from openexachangerates api
type OpenExResponse struct {
	Rates Quotes `json:"rates"`
}

func openexchangerates() error {
	statusRecordAttempt("openexchangerates")
	start := time.Now()
	var quotes OpenExResponse
	resp, err := http.Get(Config.OpenExRateLink)
	if err != nil {
		d := int64(time.Since(start) / time.Millisecond)
		err = fmt.Errorf("import failed: %w", err)
		logEvent(4, "loadOpenExchangerates", 500, err.Error(), d)
		statusRecordFailure("openexchangerates", err)
		return err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		d := int64(time.Since(start) / time.Millisecond)
		err = fmt.Errorf("read body: %w", err)
		logEvent(4, "loadOpenExchangerates", 500, err.Error(), d)
		statusRecordFailure("openexchangerates", err)
		return err
	}
	if err := json.Unmarshal(body, &quotes); err != nil {
		d := int64(time.Since(start) / time.Millisecond)
		err = fmt.Errorf("json parse: %w", err)
		logEvent(4, "loadOpenExchangerates", 500, err.Error(), d)
		statusRecordFailure("openexchangerates", err)
		return err
	}
	// Calculate USD rate. The api uses base currency USD, so to calculate other currencies rate to EUR: Rate(USD)*Rate(CurFromAPI)
	u := 1 / quotes.Rates["EUR"]
	// If USD is overided
	if Config.Stocks["USD"].Enable {
		for _, cur := range QutesinMemory {
			if cur.Category == 0 && cur.Symbol == "USD" {
				u = cur.Rate
				break
			}
		}
	}
	updated := 0
	for _, v := range strings.Split(Config.OpenExRateCurList, ",") {
		if !Config.Stocks[v].Enable {
			var r = u * quotes.Rates[v]
			if v == "EUR" {
				r = 1
			}
			str := Quote{
				Symbol:   v,
				Rate:     r,
				Category: 0,
			}
			if isElementInDB(str) {
				updateRate(str)
			} else {
				writeNewCurrency(str)
			}
			updated++
		}
	}
	for _, v := range strings.Split(Config.OpenExRateMetalList, ",") {
		var r = u * quotes.Rates[v]
		str := Quote{
			Symbol:   v,
			Rate:     r,
			Category: 2,
		}
		if isElementInDB(str) {
			updateRate(str)
		} else {
			writeNewCurrency(str)
		}
		updated++
	}
	d := int64(time.Since(start) / time.Millisecond)
	msg := fmt.Sprintf("fiat and metals updated (%d symbols)", updated)
	if Config.LogLoadRatesInfo {
		logEvent(6, "loadOpenExchangerates", 200, "Was loaded successfully", d)
	}
	statusRecordSuccess("openexchangerates", msg)
	return nil
}
