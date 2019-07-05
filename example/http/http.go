package main

import (
	"log"
	"net/http"
	"time"
	"zlimiter"
	"zlimiter/driver"
)

var limiter *zlimiter.Limits
var erro error

const (
	RATELIMITE_TOTAL = "*"
)

func Test(w http.ResponseWriter, r *http.Request) {
	bReached, left, erro := limiter.Get(RATELIMITE_TOTAL)
	if erro != nil {
		log.Println(erro.Error())
	}

	if bReached == true && left == -1 {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(http.StatusText(http.StatusTooManyRequests)))
		return
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(http.StatusText(http.StatusOK)))
		return
	}
}

func main() {

	limiter, erro = zlimiter.NewLimiter(zlimiter.LIMIT_TYPE_REDIS, driver.RedisInfo{Host: "127.0.0.1:6379", Passwd: "xxxx"})
	if erro != nil {
		log.Fatal(erro.Error())
	}
	limiter.Add(RATELIMITE_TOTAL, 10, 2*time.Second)

	http.HandleFunc("/test", Test)
	log.Fatal(http.ListenAndServe("0.0.0.0:1234", nil))
}
