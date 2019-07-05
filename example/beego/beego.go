package main

import (
	//"context"
	_ "limiterTest/routers"
	"log"
	"net/http"
	"strings"
	"time"
	"zlimiter"
	"zlimiter/driver"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
)

func IPRateLimit(l *zlimiter.Limits, ctx *context.Context) {
	remoteAddr := ctx.Request.RemoteAddr
	ips := strings.Split(remoteAddr, ":")
	// ip := ctx.Request.Header.Get("X-Forwarded-For")
	bReached, left, erro := l.Get(ips[0])
	if erro != nil {
		log.Fatal(erro.Error())
	}

	if left == -2 { // nerver add
		l.Add(ips[0], 10, 2*time.Second)
		l.Get(ips[0])
	}

	if bReached == true && left == -1 { // reach the limit
		writer := ctx.ResponseWriter
		writer.WriteHeader(http.StatusTooManyRequests)
		ctx.WriteString(http.StatusText(http.StatusTooManyRequests))
		return
	}
}

func main() {
	// create
	redisLimit, erro := zlimiter.NewLimiter(zlimiter.LIMIT_TYPE_REDIS, driver.RedisInfo{Host: "127.0.0.1:6379", Passwd: "xxx"})
	if erro != nil {
		log.Fatal(erro.Error())
	}

	beego.InsertFilter("*", beego.BeforeRouter, func(c *context.Context) {
		IPRateLimit(redisLimit, c)
	}, true)

	beego.Run()
}
