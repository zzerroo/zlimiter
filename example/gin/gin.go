package main

import (
	"log"
	"net/http"
	"strings"
	"time"
	"zlimiter/driver"

	"github.com/zzerroo/zlimiter"

	"github.com/gin-gonic/gin"
)

// Test ...
func Test(context *gin.Context) {
	context.String(http.StatusOK, http.StatusText(http.StatusTooManyRequests))
}

var limiter *zlimiter.Limits
var erro error

// Limit ...
func Limit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ipInfos := c.Request.RemoteAddr
		ips := strings.Split(ipInfos, ":")

		bReached, left, erro := limiter.Get(ips[0])
		if erro != nil {
			log.Fatal(erro.Error())
		}
		if left == -2 { // nerver add
			limiter.Add(ips[0], 10, 2*time.Second)
			limiter.Get(ips[0])
		}

		if bReached == true && left == -1 { // reach the limit
			c.Writer.WriteHeader(http.StatusTooManyRequests)
			c.Writer.WriteString(http.StatusText(http.StatusTooManyRequests))
			return
		}

		c.Next()
	}
}

func main() {
	limiter, erro = zlimiter.NewLimiter(zlimiter.LIMIT_TYPE_REDIS, driver.RedisInfo{Host: "127.0.0.1:6379", Passwd: "xxxx"})
	if erro != nil {
		log.Fatal(erro.Error())
	}

	router := gin.Default()
	router.Use(Limit())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, http.StatusText(http.StatusOK))
	})

	router.Run()
}
