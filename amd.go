package main

import (
	"strconv"

	"github.com/antchfx/htmlquery"
)

// AMD - Get Dinar rate
func AMD() {

	resp, err := htmlquery.LoadURL("https://rate.am/en/armenian-dram-exchange-rates/banks/non-cash")

	if err != nil {
		Logger2Errors("Error importing from AMD")
		return
	}
	Logger2("AZT was imported")
	t := htmlquery.FindOne(resp, "//*[@id=\"rb\"]/tbody/tr[23]/td[5]")

	r, err := strconv.ParseFloat(htmlquery.InnerText(t), 64)

	if err != nil {
		Logger2Errors("Error parsing response for AMD from string to float (Check api https://www.cba.am/en/sitepages/ExchangeArchive.aspx) ")
		return
	}

	str := Quote{
		Symbol:   "AMD",
		Rate:     r,
		Category: 0,
	}
	if isElementInDB(str) {
		updateRate(str)
	} else {
		writeNewCurrency(str)
	}
}
