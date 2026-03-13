package sources

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// UkraineMinfin implements Source for Ukrainian average fuel prices from Minfin.
// Data source: https://index.minfin.com.ua/markets/fuel/
type UkraineMinfin struct {
	URL string
}

func NewUkraineMinfin() *UkraineMinfin {
	return &UkraineMinfin{
		URL: "https://index.minfin.com.ua/markets/fuel/",
	}
}

func (s *UkraineMinfin) Name() string {
	return "ua_minfin"
}

// FetchPrices fetches average fuel prices for Ukraine.
// countryCode is expected to be "UA"; for any other code an empty result is returned.
func (s *UkraineMinfin) FetchPrices(ctx context.Context, countryCode string) ([]RawFuelPrice, error) {
	if strings.ToUpper(countryCode) != "UA" {
		return nil, nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", s.URL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ua_minfin: unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	html := string(body)

	reportDate := parseMinfinDate(html)

	// В таблице после ссылки идёт текст "А-95" с цифрами, поэтому ищем цену после </a>.
	// Один паттерн для всех видов: захватываем ключ (a96|a95|a92|dt|lpg) и цену xx,yy.
	re := regexp.MustCompile(regexp.QuoteMeta("/markets/fuel/") + `(a96|a95|a92|dt|lpg)/[^<]*</a>[^0-9]*([0-9]+,[0-9]+)`)
	matches := re.FindAllStringSubmatch(html, -1)

	nameByKey := map[string]string{
		"a96": "Бензин А-95 премиум",
		"a95": "Бензин А-95",
		"a92": "Бензин А-92",
		"dt":  "Дизель",
		"lpg": "Газ (LPG)",
	}

	var result []RawFuelPrice
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		key, priceStr := m[1], m[2]
		name, ok := nameByKey[key]
		if !ok {
			continue
		}
		priceStr = strings.ReplaceAll(priceStr, ",", ".")
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil || price <= 0 {
			continue
		}

		result = append(result, RawFuelPrice{
			Country:      "UA",
			FuelType:     name,
			Price:        price,
			Currency:     "UAH",
			SourceUpdate: reportDate,
		})
	}

	return result, nil
}

// parseMinfinDate parses the "последнее обновление: DD.MM.YYYY HH:MM" line.
func parseMinfinDate(html string) time.Time {
	re := regexp.MustCompile(`последнее обновление:\s*([0-9]{2}\.[0-9]{2}\.[0-9]{4})(?:\s+([0-9]{2}:[0-9]{2}))?`)
	m := re.FindStringSubmatch(html)
	if len(m) >= 2 {
		layout := "02.01.2006 15:04"
		dateStr := m[1]
		timePart := "00:00"
		if len(m) >= 3 && m[2] != "" {
			timePart = m[2]
		}
		ts, err := time.Parse(layout, dateStr+" "+timePart)
		if err == nil {
			return ts
		}
	}
	return time.Now()
}


