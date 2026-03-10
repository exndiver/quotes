package sources

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/xuri/excelize/v2"
)

// EUWeeklyOilBulletin implements the Source interface using the official EU XLSX
type EUWeeklyOilBulletin struct {
	LandingPage string
	BaseURL     string
	cachedData  map[string][]RawFuelPrice
	lastFetch   time.Time
	mu          sync.Mutex
}

func NewEUWeeklyOilBulletin() *EUWeeklyOilBulletin {
	return &EUWeeklyOilBulletin{
		LandingPage: "https://energy.ec.europa.eu/data-and-analysis/weekly-oil-bulletin_en",
		BaseURL:     "https://energy.ec.europa.eu",
	}
}

func (s *EUWeeklyOilBulletin) Name() string {
	return "eu_oil_bulletin"
}

// FetchPrices fetches prices for a specific country from the EU Oil Bulletin XLSX
func (s *EUWeeklyOilBulletin) FetchPrices(ctx context.Context, countryCode string) ([]RawFuelPrice, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Cache for 1 hour for debugging/testing
	if s.cachedData != nil && time.Since(s.lastFetch) < time.Hour {
		return s.cachedData[countryCode], nil
	}

	xlsxURL, err := s.findLatestXLSX(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find latest XLSX: %w", err)
	}

	log.Printf("Downloading official EU bulletin from: %s", xlsxURL)

	data, err := s.downloadAndParse(ctx, xlsxURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XLSX: %w", err)
	}

	s.cachedData = data
	s.lastFetch = time.Now()

	log.Printf("Successfully parsed fuel prices for %d countries", len(data))

	return s.cachedData[countryCode], nil
}

func (s *EUWeeklyOilBulletin) findLatestXLSX(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.LandingPage, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(`/document/download/[a-z0-9-]+_en\?filename=[^" ]*with[^" ]*Taxes[^" ]*\.xlsx`)
	match := re.FindString(string(body))
	if match == "" {
		return "", fmt.Errorf("could not find latest prices with taxes XLSX link on landing page")
	}

	if strings.HasPrefix(match, "/") {
		return s.BaseURL + match, nil
	}

	return match, nil
}

func (s *EUWeeklyOilBulletin) downloadAndParse(ctx context.Context, url string) (map[string][]RawFuelPrice, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download XLSX: %s", resp.Status)
	}

	f, err := excelize.OpenReader(resp.Body)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets in XLSX")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, err
	}

	if len(rows) < 3 {
		return nil, fmt.Errorf("unexpected XLSX structure: too few rows")
	}

	dateStr := ""
	if len(rows[1]) > 0 {
		fields := strings.Fields(rows[1][0])
		if len(fields) > 0 {
			dateStr = fields[0]
		}
	}

	reportDate := time.Now()
	if dateStr != "" {
		parsedDate, err := time.Parse("02/01/2006", dateStr)
		if err == nil {
			reportDate = parsedDate
		}
	}

	result := make(map[string][]RawFuelPrice)
	countryMapping := getEUCountryMapping()

	fuelCols := map[int]string{
		1: "Euro-super 95",
		2: "Diesel",
		6: "LPG",
	}

	for i := 2; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 2 {
			continue
		}

		countryName := strings.TrimSpace(row[0])
		code, ok := countryMapping[countryName]
		if !ok {
			continue
		}

		for colIdx, fuelType := range fuelCols {
			if len(row) <= colIdx {
				continue
			}

			rawVal := row[colIdx]
			cleanVal := cleanPriceString(rawVal)
			if cleanVal == "" {
				continue
			}

			price, err := strconv.ParseFloat(cleanVal, 64)
			if err != nil {
				log.Printf("[EU Fetcher] Failed to parse price '%s' (raw: '%s') for %s %s: %v", cleanVal, rawVal, code, fuelType, err)
				continue
			}

			// Convert to per liter (input is per 1000L)
			price = price / 1000.0

			result[code] = append(result[code], RawFuelPrice{
				Country:      code,
				FuelType:     fuelType,
				Price:        price,
				Currency:     "EUR",
				SourceUpdate: reportDate,
			})
		}
	}

	return result, nil
}

func cleanPriceString(s string) string {
	// Remove commas (thousands separator in some regions)
	s = strings.ReplaceAll(s, ",", "")
	// Keep only digits and decimal point
	return strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) || r == '.' {
			return r
		}
		return -1
	}, s)
}

func getEUCountryMapping() map[string]string {
	return map[string]string{
		"Austria":     "AT",
		"Belgium":     "BE",
		"Bulgaria":    "BG",
		"Croatia":     "HR",
		"Cyprus":      "CY",
		"Czechia":     "CZ",
		"Denmark":     "DK",
		"Estonia":     "EE",
		"Finland":     "FI",
		"France":      "FR",
		"Germany":     "DE",
		"Greece":      "GR",
		"Hungary":     "HU",
		"Ireland":     "IE",
		"Italy":       "IT",
		"Latvia":      "LV",
		"Lithuania":   "LT",
		"Luxembourg":  "LU",
		"Malta":       "MT",
		"Netherlands": "NL",
		"Poland":      "PL",
		"Portugal":    "PT",
		"Romania":     "RO",
		"Slovakia":    "SK",
		"Slovenia":    "SI",
		"Spain":       "ES",
		"Sweden":      "SE",
	}
}
