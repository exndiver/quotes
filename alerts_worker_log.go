package main

import (
	"log"
	"os"
	"sync"
)

var (
	alertsWorkerLogOnce sync.Once
	alertsWorkerLogger  *log.Logger
)

func alertsWorkerLog() *log.Logger {
	alertsWorkerLogOnce.Do(func() {
		_ = os.MkdirAll("./logs/", os.ModePerm)
		f, err := os.OpenFile("./logs/alerts_worker.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			// fallback to stderr if file can't be opened
			alertsWorkerLogger = log.New(os.Stderr, "alerts-worker ", log.LstdFlags|log.Lmicroseconds)
			alertsWorkerLogger.Printf("failed to open alerts_worker.log: %v", err)
			return
		}
		alertsWorkerLogger = log.New(f, "alerts-worker ", log.LstdFlags|log.Lmicroseconds)
	})
	return alertsWorkerLogger
}
