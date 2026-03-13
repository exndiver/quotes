package sources

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// UKGov implements Source for UK weekly road fuel prices (gov.uk XML).
// Data: Pump price in pence/litre; we convert to GBP/litre (pence/100) and service converts to USD.
type UKGov struct {
	URL string
}

func NewUKGov() *UKGov {
	return &UKGov{
		URL: "https://assets.publishing.service.gov.uk/media/69aeefc56827004e30b8a588/weekly_road_fuel_prices_090326.xml",
	}
}

func (s *UKGov) Name() string {
	return "uk_gov"
}

// FetchPrices fetches the latest weekly fuel prices for UK (GB).
// countryCode is expected to be "GB"; for any other code an empty result is returned.
func (s *UKGov) FetchPrices(ctx context.Context, countryCode string) ([]RawFuelPrice, error) {
	if strings.ToUpper(countryCode) != "GB" {
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
		return nil, fmt.Errorf("uk_gov: unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	date, petrolPence, dieselPence, err := parseUKGovXML(body)
	if err != nil {
		return nil, err
	}

	// pence/litre -> GBP/litre
	petrolGBP := petrolPence / 100
	dieselGBP := dieselPence / 100

	return []RawFuelPrice{
		{
			Country:      "GB",
			FuelType:     "Unleaded petrol",
			Price:        petrolGBP,
			Currency:     "GBP",
			SourceUpdate: date,
		},
		{
			Country:      "GB",
			FuelType:     "Diesel",
			Price:        dieselGBP,
			Currency:     "GBP",
			SourceUpdate: date,
		},
	}, nil
}

// Excel Spreadsheet ML structures; gov.uk XML uses default namespace urn:schemas-microsoft-com:office:spreadsheet
type workbook struct {
	XMLName    xml.Name    `xml:"urn:schemas-microsoft-com:office:spreadsheet Workbook"`
	Worksheets []worksheet `xml:"urn:schemas-microsoft-com:office:spreadsheet Worksheet"`
}

type worksheet struct {
	Table table `xml:"urn:schemas-microsoft-com:office:spreadsheet Table"`
}

type table struct {
	Rows []row `xml:"urn:schemas-microsoft-com:office:spreadsheet Row"`
}

type row struct {
	Cells []cell `xml:"urn:schemas-microsoft-com:office:spreadsheet Cell"`
}

type cell struct {
	Data string `xml:"urn:schemas-microsoft-com:office:spreadsheet Data"`
}

func parseUKGovXML(data []byte) (date time.Time, petrolPence, dieselPence float64, err error) {
	var w workbook
	if err := xml.Unmarshal(data, &w); err != nil {
		return time.Time{}, 0, 0, fmt.Errorf("xml unmarshal: %w", err)
	}
	return parseFromWorkbook(&w)
}

func parseFromWorkbook(w *workbook) (time.Time, float64, float64, error) {
	if len(w.Worksheets) == 0 {
		return time.Time{}, 0, 0, fmt.Errorf("no worksheets")
	}
	tbl := &w.Worksheets[0].Table
	if len(tbl.Rows) < 2 {
		return time.Time{}, 0, 0, fmt.Errorf("not enough rows")
	}
	// Last row is latest week
	last := tbl.Rows[len(tbl.Rows)-1]
	// Cells: 0=Date, 1=ULSP pump (pence), 2=ULSD pump (pence)
	if len(last.Cells) < 3 {
		return time.Time{}, 0, 0, fmt.Errorf("last row has <3 cells")
	}
	dateStr := strings.TrimSpace(last.Cells[0].Data)
	date, err := parseUKDate(dateStr)
	if err != nil {
		date = time.Now()
	}
	petrolStr := strings.TrimSpace(last.Cells[1].Data)
	dieselStr := strings.TrimSpace(last.Cells[2].Data)
	petrolPence, err1 := strconv.ParseFloat(petrolStr, 64)
	dieselPence, err2 := strconv.ParseFloat(dieselStr, 64)
	if err1 != nil || err2 != nil {
		return time.Time{}, 0, 0, fmt.Errorf("parse prices: petrol=%v diesel=%v", err1, err2)
	}
	return date, petrolPence, dieselPence, nil
}

func parseUKDate(s string) (time.Time, error) {
	// "2003-06-09T00:00:00.000" or "2026-03-09T00:00:00.000"
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "T"); idx > 0 {
		s = s[:idx]
	}
	return time.Parse("2006-01-02", s)
}
