package main

type Quote struct{
	Symbol string `json:"symbol"`
	Rate float64 `json:"rate"`
	Category int `json:"category"`
}

var Config = get_config()

func main() {
	if Config.Plugins.Exchangeratesapi == 1 {
		exchangeratesapi()
	}
}