package main

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

// Logger2 - Logger for requesting rates from external sources
func loggerAPI(str string) {

}

// Logger2Errors - errors for requesting rates from external sources
func loggerAPIErrors(str string) {
	f, err := os.OpenFile("./logs/request_logs.error.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
	log.Printf(
		"%s\t%s\t",
		time.Now(),
		str,
	)
	f.Close()
}

func loggerFatalErrors(str error) {
	f, err := os.OpenFile("./logs/fatal_logs.error.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
	log.Printf(
		"%s\t%s\t",
		time.Now(),
		str,
	)
	log.Fatal(time.Now(), " - ", str)
	f.Close()
}

func loggerErrors(str string) {
	f, err := os.OpenFile("./logs/error.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
	log.Printf(
		"%s\t%s\t",
		time.Now(),
		str,
	)
	f.Close()
}

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
func emptyReqLogger(elapsed int64, m string) {
	var l jsonLog
	l.Duration = elapsed
	l.Method = m
	l.ResponseCode = 200
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
