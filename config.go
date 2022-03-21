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
	Feedback            Feedback
	Proxy               bool
	CacheDuration       string
	LogLoadRatesInfo    bool
	Stocks              map[string]StockProps
	HistoryOldMethod    bool
	MinLogLevel         int
}

// Feedback - config for feedback
type Feedback struct {
	Type        string // telegram or googpesheet
	Googlesheet string
	Telegram    Telegram
}

//Telegram - config for feedback
type Telegram struct {
	ChatID   string
	BotToken string
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

// StockProps main properties for different stocks
type StockProps struct {
	Host     string
	Enable   bool
	Name     string
	Currency string
}

func defaultConfig() Conf {
	var Config Conf
	Config.Hosts.Mongodb = "mongodb://mongo:27017"
	Config.Hosts.Service = ":8083"
	Config.CacheDuration = "3h"
	Config.Plugins.Crypto = true
	Config.Plugins.OpenExRates = false
	Config.DownloadRates = true
	Config.LogLoadRatesInfo = false
	Config.HistoryOldMethod = true
	Config.Feedback.Type = "telegram"
	Config.MinLogLevel = 6
	return Config
}

// getConfig - loading config file
func getConfig() Conf {
	file, _ := os.Open("config/config.json")
	defer file.Close()
	decoder := json.NewDecoder(file)
	Config := defaultConfig()
	err := decoder.Decode(&Config)
	if err != nil {
		fmt.Println("error:", err)
	}
	return Config
}
