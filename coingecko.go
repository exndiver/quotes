package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// symbolToCoinGeckoID — соответствие символа криптовалюты и id в API CoinGecko
var symbolToCoinGeckoID = map[string]string{
	"BTC":  "bitcoin",
	"ETH":  "ethereum",
	"LTC":  "litecoin",
	"ETC":  "ethereum-classic",
	"XRP":  "ripple",
	"DASH": "dash",
	"ZEC":  "zcash",
	"BUSD": "binance-usd",
	"BNB":  "binancecoin",
	"ADA":  "cardano",
}

const coingeckoPriceURL = "https://api.coingecko.com/api/v3/simple/price"

// coinGeckoPriceResp — ответ API simple/price: ids -> { "usd": number }
type coinGeckoPriceResp map[string]struct {
	Usd float64 `json:"usd"`
}

// getCryptoCoinGecko загружает курсы криптовалют из CoinGecko и пишет в БД.
// В БД хранится Rate = единиц крипты за 1 EUR (как у старого CoinAPI с базой EUR),
// чтобы /api/GetRates/0/EUR возвращал «1 EUR = X BTC», а не «1 EUR = 72405 BTC».
// Цена из CoinGecko в USD переводится в «крипта за 1 EUR»: rate = usdPerEur / priceUsd.
func getCryptoCoinGecko() {
	if Config.Coinapi == nil {
		return
	}
	listStr := Config.Coinapi["list"]
	if listStr == "" {
		return
	}
	symbols := strings.Split(listStr, ",")
	var ids []string
	for _, s := range symbols {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, ok := symbolToCoinGeckoID[s]
		if !ok {
			id = strings.ToLower(s)
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return
	}

	idsParam := strings.Join(ids, ",")
	url := coingeckoPriceURL + "?ids=" + idsParam + "&vs_currencies=usd"

	start := time.Now()
	client := &http.Client{Timeout: 15 * time.Second}
	res, err := client.Get(url)
	if err != nil {
		d := int64(time.Since(start) / time.Millisecond)
		logEvent(4, "loadCryptoCoinGecko", 500, "Request error: "+err.Error(), d)
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		d := int64(time.Since(start) / time.Millisecond)
		logEvent(4, "loadCryptoCoinGecko", 500, "Read body: "+err.Error(), d)
		return
	}

	var data coinGeckoPriceResp
	if err := json.Unmarshal(body, &data); err != nil {
		d := int64(time.Since(start) / time.Millisecond)
		logEvent(4, "loadCryptoCoinGecko", 500, "JSON parse: "+err.Error(), d)
		return
	}

	// Обратная карта: coingecko id -> symbol (первый попавшийся символ для этого id)
	idToSymbol := make(map[string]string)
	for sym, id := range symbolToCoinGeckoID {
		if _, ok := idToSymbol[id]; !ok {
			idToSymbol[id] = sym
		}
	}
	for _, s := range symbols {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id := symbolToCoinGeckoID[s]
		if id == "" {
			id = strings.ToLower(s)
		}
		idToSymbol[id] = s
	}

	// Курс USD в БД = сколько USD за 1 EUR (для конвертации цены из USD в «крипта за 1 EUR»).
	usdPerEur, hasUSD := getRateFromDB("USD", 0)
	if !hasUSD || usdPerEur <= 0 {
		logEvent(4, "loadCryptoCoinGecko", 500, "USD rate not found in DB (need OpenExRates or stocks), cannot convert to EUR base", 0)
		return
	}

	saved := 0
	for id, v := range data {
		sym, ok := idToSymbol[id]
		if !ok {
			sym = strings.ToUpper(id)
		}
		if v.Usd <= 0 {
			continue
		}
		// Rate = единиц крипты за 1 EUR (1 EUR = rate BTC). price_usd за 1 монету → price_eur = price_usd/usd_per_eur → rate = 1/price_eur.
		rate := usdPerEur / v.Usd
		q := Quote{
			Symbol:   sym,
			Rate:     rate,
			Category: 1,
		}
		if isElementInDB(q) {
			updateRate(q)
		} else {
			writeNewCurrency(q)
		}
		saved++
	}

	d := int64(time.Since(start) / time.Millisecond)
	if Config.LogLoadRatesInfo {
		logEvent(6, "loadCryptoCoinGecko", 200, "Crypto loaded successfully (CoinGecko), count: "+strconv.Itoa(saved), d)
	}
}
