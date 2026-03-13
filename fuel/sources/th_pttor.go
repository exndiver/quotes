package sources

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ThailandPTTOR implements Source for Thailand retail fuel prices from PTT OR.
// Data: https://www.pttor.com/en/oil_price
// API: https://orapiweb.pttor.com/oilservice/OilPrice.asmx (SOAP CurrentOilPrice).
// Prices in THB/litre; service converts to USD.
type ThailandPTTOR struct {
	URL string
}

func NewThailandPTTOR() *ThailandPTTOR {
	return &ThailandPTTOR{
		URL: "https://orapiweb.pttor.com/oilservice/OilPrice.asmx",
	}
}

func (s *ThailandPTTOR) Name() string {
	return "th_pttor"
}

// FetchPrices fetches current oil prices for Thailand (TH).
// countryCode is expected to be "TH"; for any other code an empty result is returned.
func (s *ThailandPTTOR) FetchPrices(ctx context.Context, countryCode string) ([]RawFuelPrice, error) {
	if strings.ToUpper(countryCode) != "TH" {
		return nil, nil
	}

	soapBody := `<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Body>
    <CurrentOilPrice xmlns="http://www.pttor.com">
      <Language>en</Language>
    </CurrentOilPrice>
  </soap:Body>
</soap:Envelope>`

	req, err := http.NewRequestWithContext(ctx, "POST", s.URL, strings.NewReader(soapBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", `"https://orapiweb.pttor.com/CurrentOilPrice"`)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("th_pttor: unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	innerXML, date, err := extractCurrentOilPriceResult(body)
	if err != nil {
		return nil, err
	}

	fuels, err := parsePTTORDS(innerXML)
	if err != nil {
		return nil, err
	}

	var result []RawFuelPrice
	for _, f := range fuels {
		result = append(result, RawFuelPrice{
			Country:      "TH",
			FuelType:     f.Product,
			Price:        f.Price,
			Currency:     "THB",
			SourceUpdate: date,
		})
	}
	return result, nil
}

func extractCurrentOilPriceResult(soapBody []byte) (innerXML string, date time.Time, err error) {
	// SOAP response may use different namespaces; extract result by tag name.
	start := strings.Index(string(soapBody), "<CurrentOilPriceResult>")
	end := strings.Index(string(soapBody), "</CurrentOilPriceResult>")
	if start == -1 || end == -1 || end <= start {
		return "", time.Time{}, fmt.Errorf("CurrentOilPriceResult not found in response")
	}
	innerXML = string(soapBody)[start+len("<CurrentOilPriceResult>"):end]
	if innerXML == "" {
		return "", time.Time{}, fmt.Errorf("empty CurrentOilPriceResult")
	}
	// Result is entity-escaped XML; decode.
	innerXML = strings.ReplaceAll(innerXML, "&lt;", "<")
	innerXML = strings.ReplaceAll(innerXML, "&gt;", ">")
	innerXML = strings.ReplaceAll(innerXML, "&amp;", "&")
	innerXML = strings.ReplaceAll(innerXML, "&quot;", `"`)
	innerXML = strings.ReplaceAll(innerXML, "&apos;", "'")

	// Parse inner XML to get first PRICE_DATE for SourceUpdate.
	var ds pttorDS
	if err := xml.Unmarshal([]byte(innerXML), &ds); err != nil {
		return "", time.Time{}, fmt.Errorf("inner xml: %w", err)
	}
	if len(ds.Fuels) == 0 {
		return "", time.Time{}, fmt.Errorf("no FUEL elements")
	}
	date, _ = parsePTTORDate(ds.Fuels[0].PriceDate)
	if date.IsZero() {
		date = time.Now()
	}
	return innerXML, date, nil
}

type pttorDS struct {
	XMLName xml.Name   `xml:"PTTOR_DS"`
	Fuels   []pttorFuel `xml:"FUEL"`
}

type pttorFuel struct {
	PriceDate string  `xml:"PRICE_DATE"`
	Product   string  `xml:"PRODUCT"`
	Price     float64 `xml:"PRICE"`
}

func parsePTTORDS(innerXML string) ([]pttorFuel, error) {
	var ds pttorDS
	if err := xml.Unmarshal([]byte(innerXML), &ds); err != nil {
		return nil, err
	}
	return ds.Fuels, nil
}

// parsePTTORDate parses "2026-03-10T05:00" to time.Time.
func parsePTTORDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if len(s) >= 16 {
		return time.Parse("2006-01-02T15:04", s[:16])
	}
	if len(s) >= 10 {
		return time.Parse("2006-01-02", s[:10])
	}
	return time.Time{}, fmt.Errorf("invalid date: %s", s)
}

