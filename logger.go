package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Logger1 - Access logs
func logger_access(r *http.Request) {
	f, err := os.OpenFile("./logs/logs.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
	log.Printf(
		"%s\t-\t%s\t%s\t%s\t%s\t",
		strings.Join(r.Header["X-Forwarded-For"], ","),
		r.Method,
		r.RequestURI,
		r.Header,
		r.RemoteAddr,
	)
	f.Close()
}

// Logger1Errors - Errors logs for api request
func logger_access_errors(str string) {
	f, err := os.OpenFile("./logs/logs.error.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
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

// Logger2 - Logger for requesting rates from external sources
func loggerApi(str string) {
	/*f, err := os.OpenFile("./logs/request_logs.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
	log.Printf(
		"%s\t%s\t",
		time.Now(),
		str,
	)
	f.Close()*/
}

// Logger2Errors - errors for requesting rates from external sources
func loggerApi_errors(str string) {
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
	log.Fatal(
		"%s\t%s\t",
		time.Now(),
		str,
	)
	f.Close()
}
