package main
 
import (
    "log"
	"net/http"
	"os"
	"time"
)

func Logger1(r *http.Request){
	f, err := os.OpenFile("./logs/logs.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
    log.Printf(
            "%s\t%s\t%s\t%s\t",
            r.Method,
            r.RequestURI,
            r.Header,
            r.RemoteAddr,
			  )
	f.Close()
}

func Logger2(str string){
	f, err := os.OpenFile("./logs/request_logs.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
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

func Logger2Errors(str string){
	f, err := os.OpenFile("./logs/request_logs.error.log", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
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