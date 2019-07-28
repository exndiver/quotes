package main

import (
	"strconv"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

// KZT - Get Dinar rate
func KZT() {
	resp, err := htmlquery.LoadURL("https://nationalbank.kz/?furl=cursFull&switch=eng")

	if err != nil {
		Logger2Errors("Error importing from Srb bank")
		return
	}
	Logger2("Srb was imported")
	var a *html.Node
	for _, n := range htmlquery.Find(resp, "/html/body/table/tbody/tr[3]/td/table/tbody/tr/td[3]/div[2]/form/table/tbody/tr") {
		c := htmlquery.FindOne(n, "//td[3]")
		if htmlquery.InnerText(c) == "EUR / KZT" {
			a = htmlquery.FindOne(n, "//td[4]")
			break
		}
	}
	if a == nil {
		Logger2Errors("Error reading response for KZT (Check api https://nationalbank.kz/?furl=cursFull&switch=eng) ")
		return
	}

	r, err := strconv.ParseFloat(htmlquery.InnerText(a), 64)

	if err != nil {
		Logger2Errors("Error parsing response for KZT from string to float (Check api https://nationalbank.kz/?furl=cursFull&switch=eng) ")
		return
	}

	str := Quote{
		Symbol:   "KZT",
		Rate:     r,
		Category: 0,
	}
	if isElementInDB(str) {
		updateRate(str)
	} else {
		writeNewCurrency(str)
	}
}
