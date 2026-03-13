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

// USAGlobalPetrolPrices implements Source for USA retail fuel from GlobalPetrolPrices.com.
// Data: https://www.globalpetrolprices.com/USA/
// Table "Fuels, price per liter" — prices already in USD; no conversion.
type USAGlobalPetrolPrices struct {
	URL string
}

func NewUSAGlobalPetrolPrices() *USAGlobalPetrolPrices {
	return &USAGlobalPetrolPrices{
		URL: "https://www.globalpetrolprices.com/USA/",
	}
}

func (s *USAGlobalPetrolPrices) Name() string {
	return "us_globalpetrolprices"
}

// FetchPrices fetches fuel prices for USA (US).
func (s *USAGlobalPetrolPrices) FetchPrices(ctx context.Context, countryCode string) ([]RawFuelPrice, error) {
	if strings.ToUpper(countryCode) != "US" {
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
		return nil, fmt.Errorf("us_globalpetrolprices: unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	html := string(body)

	// Same table structure as Russia: indicatorName, date, first value column (USD for USA).
	re := regexp.MustCompile(`<a class="indicatorName"[^>]*>\s*([^<]+?)\s*</a>\s*</th>\s*<td class="value">\s*([0-9]{2}\.[0-9]{2}\.[0-9]{4})\s*</td>\s*<td class="value">\s*([0-9]+\.[0-9]+)\s*</td>`)
	matches := re.FindAllStringSubmatch(html, -1)

	if len(matches) == 0 {
		return nil, fmt.Errorf("us_globalpetrolprices: no fuel rows found")
	}

	var result []RawFuelPrice
	for _, m := range matches {
		if len(m) < 4 {
			continue
		}
		name := strings.TrimSpace(m[1])
		dateStr := strings.TrimSpace(m[2])
		priceStr := strings.TrimSpace(m[3])
		if strings.Contains(strings.ToLower(name), "electricity") ||
			strings.Contains(strings.ToLower(name), "natural gas") ||
			strings.Contains(strings.ToLower(name), "household") {
			continue
		}
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil || name == "" {
			continue
		}
		date, err := time.Parse("02.01.2006", dateStr)
		if err != nil {
			date = time.Now()
		}
		result = append(result, RawFuelPrice{
			Country:      "US",
			FuelType:     name,
			Price:        price,
			Currency:     "USD",
			SourceUpdate: date,
		})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("us_globalpetrolprices: parsed zero prices")
	}
	return result, nil
}
