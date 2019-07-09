package main

import (
	"encoding/json"
  	"os"
	"fmt"
)

type Conf struct{
	Hosts Hosts
	Service string
	Mongodb string
	AvialibleTypes string
	AvialibleList CurrenciesType
	Plugins Plugins
}

type Hosts struct{
	Service string
	Mongodb string
}

type CurrenciesType struct{
	Currencies string
}
type Plugins struct{
	Exchangeratesapi bool
}

func get_config() Conf{
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