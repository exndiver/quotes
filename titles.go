package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type title map[string]string

type locale map[string]title

func loadLocales() locale {
	file, _ := os.Open("titles.json")
	defer file.Close()
	decoder := json.NewDecoder(file)
	var l locale
	err := decoder.Decode(&l)
	if err != nil {
		fmt.Println("error:", err)
	}
	return l
}

func getLocale(l string) []byte {
	if len(Locales[l]) == 0 {
		l = Config.DefaultLocale
	}

	jsonResult, _ := json.Marshal(Locales[l])
	return jsonResult
}
