package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func stockRate() error {
	if !stocksEnabled() {
		return nil
	}
	statusRecordAttempt("stocks")
	start := time.Now()
	var errs []string
	var updated []string

	for _, v := range Config.Stocks {
		if !v.Enable {
			continue
		}
		resp, err := http.Get(v.Host)
		if err != nil {
			d := int64(time.Since(start) / time.Millisecond)
			msg := fmt.Sprintf("%s: import failed: %v", v.Currency, err)
			logEvent(4, "loadStocks", 500, msg, d)
			errs = append(errs, msg)
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			d := int64(time.Since(start) / time.Millisecond)
			msg := fmt.Sprintf("%s: read body: %v", v.Currency, err)
			logEvent(4, "loadStocks", 500, msg, d)
			errs = append(errs, msg)
			continue
		}
		q := 0.0
		ok := true
		switch v.Name {
		case "EU":
			q, ok = getEURates(v.Currency, body)
			if !ok {
				d := int64(time.Since(start) / time.Millisecond)
				msg := fmt.Sprintf("%s: parse failed from %s", v.Currency, v.Host)
				logEvent(4, "loadStocks", 500, msg, d)
				errs = append(errs, msg)
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
		updated = append(updated, v.Currency)
		d := int64(time.Since(start) / time.Millisecond)
		if Config.LogLoadRatesInfo {
			logEvent(6, "loadStocks", 200, v.Host+" was loaded successfully", d)
		}
	}

	if len(updated) == 0 && len(errs) > 0 {
		err := fmt.Errorf(strings.Join(errs, "; "))
		statusRecordFailure("stocks", err)
		return err
	}
	msg := "stocks updated: " + strings.Join(updated, ", ")
	if len(errs) > 0 {
		err := fmt.Errorf("%s; failures: %s", msg, strings.Join(errs, "; "))
		statusRecordFailure("stocks", err)
		return err
	}
	statusRecordSuccess("stocks", msg)
	return nil
}
