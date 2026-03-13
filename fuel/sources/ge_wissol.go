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

// GeorgiaWissol implements Source for Georgia retail fuel prices from Wissol.
// Data: https://wissol.ge/en/ (block .top-prices_item — names + price in ₾).
// ₾ is Georgian Lari (GEL); service converts to USD via GetRate("GEL").
type GeorgiaWissol struct {
	URL string
}

func NewGeorgiaWissol() *GeorgiaWissol {
	return &GeorgiaWissol{
		URL: "https://wissol.ge/en/",
	}
}

func (s *GeorgiaWissol) Name() string {
	return "ge_wissol"
}

// FetchPrices fetches fuel prices for Georgia (GE).
func (s *GeorgiaWissol) FetchPrices(ctx context.Context, countryCode string) ([]RawFuelPrice, error) {
	if strings.ToUpper(countryCode) != "GE" {
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
		return nil, fmt.Errorf("ge_wissol: unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	html := string(body)

	// Each item: first <p style=...> contains fuel name, next <p> has "3.69" then newline and "₾"
	re := regexp.MustCompile(`(?s)<div class="top-prices_item">.*?<p style="[^"]*">\s*([^<]+?)\s*</p>\s*<p>\s*([0-9]+\.[0-9]+)\s*₾`)
	matches := re.FindAllStringSubmatch(html, -1)

	if len(matches) == 0 {
		return nil, fmt.Errorf("ge_wissol: no top-prices_item blocks found")
	}

	now := time.Now()
	var result []RawFuelPrice
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		name := strings.TrimSpace(m[1])
		price, err := strconv.ParseFloat(strings.TrimSpace(m[2]), 64)
		if err != nil || name == "" {
			continue
		}
		result = append(result, RawFuelPrice{
			Country:      "GE",
			FuelType:     name,
			Price:        price,
			Currency:     "GEL",
			SourceUpdate: now,
		})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("ge_wissol: parsed zero prices")
	}
	return result, nil
}
