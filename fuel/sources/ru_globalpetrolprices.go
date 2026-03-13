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

// RussiaGlobalPetrolPrices implements Source for Russia retail fuel from GlobalPetrolPrices.com.
// Data: https://www.globalpetrolprices.com/Russia/
// Table "Fuels, price per liter" — RUB per liter; service converts to USD.
type RussiaGlobalPetrolPrices struct {
	URL string
}

func NewRussiaGlobalPetrolPrices() *RussiaGlobalPetrolPrices {
	return &RussiaGlobalPetrolPrices{
		URL: "https://www.globalpetrolprices.com/Russia/",
	}
}

func (s *RussiaGlobalPetrolPrices) Name() string {
	return "ru_globalpetrolprices"
}

// FetchPrices fetches fuel prices for Russia (RU).
func (s *RussiaGlobalPetrolPrices) FetchPrices(ctx context.Context, countryCode string) ([]RawFuelPrice, error) {
	if strings.ToUpper(countryCode) != "RU" {
		return nil, nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", s.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; quotes-fuel/1.0)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ru_globalpetrolprices: unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	html := string(body)

	// Only the fuel table: rows with indicatorName + date + RUB column (skip electricity/gas tables).
	// <a class="indicatorName" ...>Gasoline prices</a></th><td class="value">09.03.2026</td><td class="value">66.81</td>
	re := regexp.MustCompile(`<a class="indicatorName"[^>]*>\s*([^<]+?)\s*</a>\s*</th>\s*<td class="value">\s*([0-9]{2}\.[0-9]{2}\.[0-9]{4})\s*</td>\s*<td class="value">\s*([0-9]+\.[0-9]+)\s*</td>`)
	matches := re.FindAllStringSubmatch(html, -1)

	if len(matches) == 0 {
		return nil, fmt.Errorf("ru_globalpetrolprices: no fuel rows found")
	}

	var result []RawFuelPrice
	for _, m := range matches {
		if len(m) < 4 {
			continue
		}
		name := strings.TrimSpace(m[1])
		dateStr := strings.TrimSpace(m[2])
		rubStr := strings.TrimSpace(m[3])
		// Skip non-fuel rows if any slip through (e.g. link text containing "electricity")
		if strings.Contains(strings.ToLower(name), "electricity") ||
			strings.Contains(strings.ToLower(name), "natural gas") ||
			strings.Contains(strings.ToLower(name), "household") {
			continue
		}
		price, err := strconv.ParseFloat(rubStr, 64)
		if err != nil || name == "" {
			continue
		}
		date, err := time.Parse("02.01.2006", dateStr)
		if err != nil {
			date = time.Now()
		}
		result = append(result, RawFuelPrice{
			Country:      "RU",
			FuelType:     name,
			Price:        price,
			Currency:     "RUB",
			SourceUpdate: date,
		})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("ru_globalpetrolprices: parsed zero prices")
	}
	return result, nil
}
