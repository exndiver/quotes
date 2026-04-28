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
	Coinapi             map[string]string
	OpenExRateCurList   string
	OpenExRateMetalList string
	OpenExRateLink      string
	Feedback            Feedback
	Proxy               bool
	CacheDuration       string
	LogLoadRatesInfo    bool
	Stocks              map[string]StockProps
	MinLogLevel         int
	AlertsActiveLimit   int           `json:"alerts_active_limit"`
	RateMinStep         float64       `json:"rate_min_step"`
	AlertsWorkers       AlertsWorkers `json:"alerts_workers"`
	Firebase            Firebase      `json:"firebase"`
}

// Feedback - config for feedback
type Feedback struct {
	Type        string // telegram or googpesheet
	Googlesheet string
	Telegram    Telegram
}

// Telegram - config for feedback
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

type AlertsWorkers struct {
	Enabled            bool   `json:"enabled"`
	ScheduleInterval   string `json:"schedule_interval"`
	ThresholdInterval  string `json:"threshold_interval"`
	ScheduleBatchSize  int    `json:"schedule_batch_size"`
	ThresholdBatchSize int    `json:"threshold_batch_size"`
}

type Firebase struct {
	CredentialsFile string `json:"credentials_file"`
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
	Config.Feedback.Type = "telegram"
	Config.MinLogLevel = 6
	Config.AlertsActiveLimit = 100
	Config.RateMinStep = 0.01
	Config.AlertsWorkers.Enabled = true
	Config.AlertsWorkers.ScheduleInterval = "5m"
	Config.AlertsWorkers.ThresholdInterval = "30s"
	Config.AlertsWorkers.ScheduleBatchSize = 50
	Config.AlertsWorkers.ThresholdBatchSize = 100
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
