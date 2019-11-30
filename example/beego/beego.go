package main

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
	"github.com/zzerroo/zlimiter"
)

// IPRateLimit ..
func IPRateLimit(l *zlimiter.Limits, ctx *context.Context) {
	remoteAddr := ctx.Request.RemoteAddr
	ips := strings.Split(remoteAddr, ":")
	// ip := ctx.Request.Header.Get("X-Forwarded-For")
	left, erro := l.Get(ips[0])
	if erro != nil {
		log.Fatal(erro.Error())
	}

	writer := ctx.ResponseWriter
	if left == zlimiter.ErrorReturnItemNotExist { // nerver add
		l.Add(ips[0], 10, 1*time.Second)
		l.Get(ips[0])

		writer.WriteHeader(http.StatusOK)
		ctx.WriteString(http.StatusText(http.StatusOK))
	} else if left == -1 { // reach the limit
		writer.WriteHeader(http.StatusTooManyRequests)
		ctx.WriteString(http.StatusText(http.StatusTooManyRequests))
	} else {
		writer.WriteHeader(http.StatusOK)
		ctx.WriteString(http.StatusText(http.StatusOK))
	}

	return
}

func main() {
	// create
	redisLimit := zlimiter.NewLimiter(zlimiter.LimitMemFixWindow)
	beego.InsertFilter("*", beego.BeforeRouter, func(c *context.Context) {
		IPRateLimit(redisLimit, c)
	}, true)

	beego.Run()
}
