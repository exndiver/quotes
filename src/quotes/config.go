package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// Conf - main config struct
type Conf struct {
	Hosts          Hosts
	Service        string
	Mongodb        string
	AvialibleTypes string
	AvialibleList  map[string]string
	Plugins        Plugins
	Cryptoapilist  map[string]string
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
	Exchangeratesapi bool
	Crypto           bool
	Blrd             bool
	Srb              bool
	Ukr              bool
	Kzt              bool
	Azt              bool
	Amd              bool
	Gel              bool
}

// getConfig - loading config file
func getConfig() Conf {
	file, _ := os.Open("config.json")
	defer file.Close()
	decoder := json.NewDecoder(file)
	var Config Conf
	err := decoder.Decode(&Config)
	if err != nil {
		fmt.Println("error:", err)
	}
	return Config
}
