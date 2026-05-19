package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGPPFetchPrices_integerPrice(t *testing.T) {
	html := `<table>
<tr><td class="value">Fuels, price per liter</td><td class="value">Date</td><td class="value">AFN</td></tr>
<tr><th class="th"><a class="indicatorName" href='/Afghanistan/gasoline_prices/'>Gasoline prices</a></th>
<td class="value">11.05.2026</td><td class="value">63</td></tr>
<tr><th class="th"><a class="indicatorName" href='/Afghanistan/diesel_prices/'>Diesel prices</a></th>
<td class="value">11.05.2026</td><td class="value">58</td></tr>
</table>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(html))
	}))
	defer srv.Close()

	gpp := &GPP{BaseURL: srv.URL}
	prices, err := gpp.FetchPrices(context.Background(), "AF")
	if err != nil {
		t.Fatalf("FetchPrices: %v", err)
	}
	if len(prices) != 2 {
		t.Fatalf("expected 2 prices, got %d", len(prices))
	}
	if prices[0].Price != 63 || prices[0].Currency != "AFN" {
		t.Fatalf("unexpected first price: %+v", prices[0])
	}
}
