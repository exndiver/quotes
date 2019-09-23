package main

import (
	"strconv"
	"strings"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

// GEL - Get Dinar rate
func GEL() {

	resp, err := htmlquery.LoadURL("https://www.nbg.gov.ge/index.php?m=582&lng=eng")

	if err != nil {
		Logger2Errors("Error importing from GEL")
		return
	}
	Logger2("GEL was imported")
	var a *html.Node
	for _, n := range htmlquery.Find(resp, "//*[@id=\"currency_id\"]/table/tbody/tr") {
		c := htmlquery.FindOne(n, "//td[1]")

		if strings.Contains(htmlquery.InnerText(c), "EUR") {
			a = htmlquery.FindOne(n, "//td[3]")
			break
		}
	}
	if a == nil {
		Logger2Errors("Error reading response for GEL (Check api https://www.nbg.gov.ge/index.php?m=582&lng=eng) ")
		return
	}

	r, err := strconv.ParseFloat(strings.TrimSpace(htmlquery.InnerText(a)), 64)

	if err != nil {
		Logger2Errors("Error parsing response for GEL from string to float (Check api https://www.nbg.gov.ge/index.php?m=582&lng=eng) ")
		return
	}

	str := Quote{
		Symbol:   "GEL",
		Rate:     r,
		Category: 0,
	}
	if isElementInDB(str) {
		updateRate(str)
	} else {
		writeNewCurrency(str)
	}
}
