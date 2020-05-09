package main

// level:
//    6 - info
//    5 - notice
//    4 - Warning
//		3 - Error
//		2 - Critical

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

type jsonLog struct {
	Version              string    `json:"version"`
	Host                 string    `json:"host"`
	Date                 time.Time `json:"_date"`
	Level                int       `json:"level"`
	Method               string    `json:"short_message"`
	RequestURI           string    `json:"_requestURI"`
	RequestHeaders       string    `json:"_RequestHeaders"`
	Request              string    `json:"_request"`
	RequestRemoteAddress string    `json:"_requestRemoteAddress"`
	Response             string    `json:"_response"`
	ResponseCode         int       `json:"_responseCode"`
	Duration             int64     `json:"_duration"`
}

//Simple log
func logEvent(lv int, m string, rc int, r string, d int64) {
	var l jsonLog
	l.Level = lv
	l.Method = m
	l.ResponseCode = rc
	l.Response = r
	l.Duration = d
	loggerJSON(l)
}

func logError(m string, r string, lv int) {
	var l jsonLog
	l.Level = lv
	l.Method = m
	l.ResponseCode = 0
	l.Response = r
	l.Duration = 0
	loggerJSON(l)
}

//Log Data to json file
func loggerJSON(l jsonLog) {
	l.Date = time.Now()
	if l.Level == 0 {
		l.Level = 6
	}
	if l.Version == "" {
		l.Version = "1.1"
	}
	if l.Host == "" {
		l.Host = "Quotes"
	}
	if l.ResponseCode == 0 {
		l.ResponseCode = 200
	}
	f, err := os.OpenFile("./logs/logs.json", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	data, _ := json.Marshal(l)
	f.WriteString(string(data) + "\n")
	f.Close()
}
