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

// GPP implements Source for any country from GlobalPetrolPrices.com.
// URL format: https://www.globalpetrolprices.com/{slug}/ (slug with hyphens).
// Currency is parsed from the fuel table header; use request_delay_seconds in config to throttle.
var gppCountrySlug = map[string]string{
	"AE": "United-Arab-Emirates", "AF": "Afghanistan", "AL": "Albania", "AD": "Andorra", "AO": "Angola",
	"AR": "Argentina", "AM": "Armenia", "AW": "Aruba", "AU": "Australia", "AT": "Austria", "AZ": "Azerbaijan",
	"BS": "Bahamas", "BH": "Bahrain", "BD": "Bangladesh", "BB": "Barbados", "BY": "Belarus", "BE": "Belgium",
	"BZ": "Belize", "BJ": "Benin", "BM": "Bermuda", "BT": "Bhutan", "BO": "Bolivia", "BA": "Bosnia-and-Herzegovina",
	"BW": "Botswana", "BR": "Brazil", "BN": "Brunei", "BG": "Bulgaria", "BF": "Burkina-Faso", "MM": "Burma",
	"BI": "Burundi", "KH": "Cambodia", "CM": "Cameroon", "CA": "Canada", "CV": "Cape-Verde",
	"CF": "Central-African-Republic", "TD": "Chad", "CL": "Chile", "CN": "China", "CO": "Colombia",
	"KM": "Comoros", "CG": "Congo", "CD": "Democratic-Republic-of-the-Congo", "CR": "Costa-Rica",
	"CI": "Ivory-Coast", "HR": "Croatia", "CU": "Cuba", "CW": "Curacao", "CY": "Cyprus", "CZ": "Czech-Republic",
	"DK": "Denmark", "DJ": "Djibouti", "DM": "Dominica", "DO": "Dominican-Republic", "EC": "Ecuador",
	"EG": "Egypt", "SV": "El-Salvador", "GQ": "Equatorial-Guinea", "ER": "Eritrea", "EE": "Estonia",
	"ET": "Ethiopia", "FJ": "Fiji", "FI": "Finland", "FR": "France", "GA": "Gabon", "GM": "Gambia",
	"GE": "Georgia", "DE": "Germany", "GH": "Ghana", "GR": "Greece", "GD": "Grenada", "GT": "Guatemala",
	"GN": "Guinea", "GW": "Guinea-Bissau", "GY": "Guyana", "HT": "Haiti", "HN": "Honduras",
	"HK": "Hong-Kong", "HU": "Hungary", "IS": "Iceland", "IN": "India", "ID": "Indonesia", "IR": "Iran",
	"IQ": "Iraq", "IE": "Ireland", "IL": "Israel", "IT": "Italy", "JM": "Jamaica", "JP": "Japan",
	"JO": "Jordan", "KZ": "Kazakhstan", "KE": "Kenya", "KW": "Kuwait", "KG": "Kyrgyzstan", "LA": "Laos",
	"LV": "Latvia", "LB": "Lebanon", "LS": "Lesotho", "LR": "Liberia", "LY": "Libya", "LI": "Liechtenstein",
	"LT": "Lithuania", "LU": "Luxembourg", "MO": "Macao", "MK": "Macedonia", "MG": "Madagascar",
	"MW": "Malawi", "MY": "Malaysia", "MV": "Maldives", "ML": "Mali", "MT": "Malta", "MR": "Mauritania",
	"MU": "Mauritius", "MX": "Mexico", "MD": "Moldova", "MC": "Monaco", "MN": "Mongolia", "ME": "Montenegro",
	"MA": "Morocco", "MZ": "Mozambique", "NA": "Namibia", "NP": "Nepal", "NL": "Netherlands",
	"NZ": "New-Zealand", "NI": "Nicaragua", "NE": "Niger", "NG": "Nigeria", "NO": "Norway", "OM": "Oman",
	"PK": "Pakistan", "PA": "Panama", "PG": "Papua-New-Guinea", "PY": "Paraguay", "PE": "Peru",
	"PH": "Philippines", "PL": "Poland", "PT": "Portugal", "PR": "Puerto-Rico", "QA": "Qatar",
	"RO": "Romania", "RU": "Russia", "RW": "Rwanda", "KN": "Saint-Kitts-and-Nevis", "LC": "Saint-Lucia",
	"VC": "Saint-Vincent-and-the-Grenadines", "WS": "Samoa", "SM": "San-Marino", "ST": "Sao-Tome-and-Principe",
	"SA": "Saudi-Arabia", "SN": "Senegal", "RS": "Serbia", "SC": "Seychelles", "SL": "Sierra-Leone",
	"SG": "Singapore", "SK": "Slovakia", "SI": "Slovenia", "SO": "Somalia", "ZA": "South-Africa",
	"KR": "South-Korea", "SS": "South-Sudan", "ES": "Spain", "LK": "Sri-Lanka", "SD": "Sudan",
	"SR": "Suriname", "SZ": "Swaziland", "SE": "Sweden", "CH": "Switzerland", "SY": "Syria", "TW": "Taiwan",
	"TJ": "Tajikistan", "TZ": "Tanzania", "TH": "Thailand", "TL": "Timor-Leste", "TG": "Togo",
	"TT": "Trinidad-and-Tobago", "TN": "Tunisia", "TR": "Turkey", "TM": "Turkmenistan", "UG": "Uganda",
	"UA": "Ukraine", "GB": "United-Kingdom", "US": "USA", "UY": "Uruguay", "UZ": "Uzbekistan",
	"VU": "Vanuatu", "VE": "Venezuela", "VN": "Vietnam", "YE": "Yemen", "ZM": "Zambia", "ZW": "Zimbabwe",
	"KY": "Cayman-Islands", "YT": "Mayotte", "WF": "Wallis-and-Futuna",
}

// GPP is the generic GlobalPetrolPrices.com source.
type GPP struct {
	BaseURL string
}

func NewGPP() *GPP {
	return &GPP{BaseURL: "https://www.globalpetrolprices.com"}
}

func (s *GPP) Name() string {
	return "gpp"
}

func (s *GPP) FetchPrices(ctx context.Context, countryCode string) ([]RawFuelPrice, error) {
	code := strings.ToUpper(countryCode)
	slug, ok := gppCountrySlug[code]
	if !ok {
		return nil, nil
	}

	url := s.BaseURL + "/" + slug + "/"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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
		return nil, fmt.Errorf("gpp: %s returned %s", url, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	html := string(body)

	// Currency: first value column header after "Date" in the fuel table (e.g. RUB, USD, EUR).
	currencyRe := regexp.MustCompile(`Fuels, price per liter</td>\s*<td class="value">\s*Date\s*</td>\s*<td class="value">\s*([A-Z]{2,3})\s*</td>`)
	currencyMatch := currencyRe.FindStringSubmatch(html)
	currency := "USD"
	if len(currencyMatch) >= 2 {
		currency = strings.TrimSpace(currencyMatch[1])
	}

	// Rows: indicatorName, date, first value
	re := regexp.MustCompile(`<a class="indicatorName"[^>]*>\s*([^<]+?)\s*</a>\s*</th>\s*<td class="value">\s*([0-9]{2}\.[0-9]{2}\.[0-9]{4})\s*</td>\s*<td class="value">\s*([0-9]+\.[0-9]+)\s*</td>`)
	matches := re.FindAllStringSubmatch(html, -1)

	if len(matches) == 0 {
		return nil, fmt.Errorf("gpp: no fuel rows at %s", url)
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
			strings.Contains(strings.ToLower(name), "household") ||
			strings.Contains(strings.ToLower(name), "business") {
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
			Country:      code,
			FuelType:     name,
			Price:        price,
			Currency:     currency,
			SourceUpdate: date,
		})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("gpp: parsed zero fuel prices at %s", url)
	}
	return result, nil
}
