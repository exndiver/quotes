package main

import (
	"strconv"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

// AZT - Get Dinar rate
func AZT() {

	resp, err := htmlquery.LoadURL("https://www.cbar.az/currency/rates")

	if err != nil {
		Logger2Errors("Error importing from AZT")
		return
	}
	Logger2("AZT was imported")
	var a *html.Node
	for _, n := range htmlquery.Find(resp, "/html/body/div/div[3]/div/div/div/div[3]/div[2]/div/div[2]/div") {
		c := htmlquery.FindOne(n, "//div[2]")

		if htmlquery.InnerText(c) == "eur" {
			a = htmlquery.FindOne(n, "//div[3]")
			break
		}
	}
	if a == nil {
		Logger2Errors("Error reading response for AZT (Check api https://nationalbank.kz/?furl=cursFull&switch=eng) ")
		return
	}

	r, err := strconv.ParseFloat(htmlquery.InnerText(a), 64)

	if err != nil {
		Logger2Errors("Error parsing response for AZT from string to float (Check api https://nationalbank.kz/?furl=cursFull&switch=eng) ")
		return
	}

	str := Quote{
		Symbol:   "AZN",
		Rate:     r,
		Category: 0,
	}
	if isElementInDB(str) {
		updateRate(str)
	} else {
		writeNewCurrency(str)
	}
}
