package main

import (
	"encoding/json"
  	"os"
	"fmt"
)

type Conf struct{
	Hosts Hosts
	Plugins Plugins
}

type Hosts struct{
	Mongodb string
}

type Plugins struct{
	Exchangeratesapi int
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