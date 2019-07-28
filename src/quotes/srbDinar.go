package main

import (
	"strconv"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

// SrbDinar - Get Dinar rate
func SrbDinar() {
	resp, err := htmlquery.LoadURL("https://www.nbs.rs/kursnaListaModul/srednjiKurs.faces?lang=lat")

	if err != nil {
		Logger2Errors("Error importing from Srb bank")
	}
	Logger2("Srb was imported")
	var a *html.Node
	for _, n := range htmlquery.Find(resp, "//*[@id=\"index:srednjiKursList:tbody_element\"]/tr") {
		c := htmlquery.FindOne(n, "//td[3]")
		if htmlquery.InnerText(c) == "EUR" {
			a = htmlquery.FindOne(n, "//td[5]")
			break
		}
	}
	if a == nil {
		Logger2Errors("Error reading response for SRB (Check api https://www.nbs.rs/kursnaListaModul/srednjiKurs.faces?lang=lat) ")
		return
	}

	r, err := strconv.ParseFloat(htmlquery.InnerText(a), 64)

	if err != nil {
		Logger2Errors("Error parsing response for SRB from string to float (Check api https://www.nbs.rs/kursnaListaModul/srednjiKurs.faces?lang=lat) ")
		return
	}

	str := Quote{
		Symbol:   "RSD",
		Rate:     r,
		Category: 0,
	}
	if isElementInDB(str) {
		updateRate(str)
	} else {
		writeNewCurrency(str)
	}
}
