package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Conf - main config struct
type Conf struct {
	Hosts               Hosts
	Service             string
	Mongodb             string
	AvialibleTypes      string
	AvialibleList       map[string]string
	DownloadRates       bool
	Plugins             Plugins
	DefaultLocale       string
	Cryptoapilist       map[string]string
	OpenExRateCurList   string
	OpenExRateMetalList string
	OpenExRateLink      string
	Feedback            string
	Proxy               bool
}

// Hosts - hosts configurations
type Hosts struct {
	Service string
	Mongodb string
}

// CurrenciesType - list of currencies type
type CurrenciesType struct {
	Currencies string
}

// Plugins - which types should be used
type Plugins struct {
	Crypto      bool
	OpenExRates bool
}

// getConfig - loading config file
func getConfig() Conf {
	file, _ := os.Open("config/config.json")
	defer file.Close()
	decoder := json.NewDecoder(file)
	var Config Conf
	Config.DownloadRates = true
	err := decoder.Decode(&Config)
	if err != nil {
		fmt.Println("error:", err)
	}
	return Config
}
