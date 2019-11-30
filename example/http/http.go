package main

import (
	"github.com/zzerroo/zlimiter"
	"log"
	"net/http"
	"time"
)

var limiter *zlimiter.Limits
var erro error

const (
	RATELIMITE_TOTAL = "*"
)

func Test(w http.ResponseWriter, r *http.Request) {
	left, erro := limiter.Get(RATELIMITE_TOTAL)
	if erro != nil {
		log.Println(erro.Error())
	}

	if left == -1 {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(http.StatusText(http.StatusTooManyRequests)))
		return
	} else if left >= 0 && erro != nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(http.StatusText(http.StatusOK)))
		return
	}
}

func main() {

	limiter = zlimiter.NewLimiter(zlimiter.LimitMemFixWindow)
	limiter.Add(RATELIMITE_TOTAL, 10, 1*time.Second)

	http.HandleFunc("/test", Test)
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
}
