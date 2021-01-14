package main

import (
	"log"
	"net/http"
	"strings"
	"time"

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

		left, erro := limiter.Get(ips[0])
		if left == zlimiter.ErrorReturnItemNotExist { // nerver add
			limiter.Add(ips[0], 10, 1*time.Second)
			limiter.Get(ips[0])
		}

		if left == -1 { // reach the limit
			c.Writer.WriteHeader(http.StatusTooManyRequests)
			c.Writer.WriteString(http.StatusText(http.StatusTooManyRequests))
			c.Abort()
			return
		}

		if left >= 0 && erro != nil {
			c.Writer.WriteHeader(http.StatusOK)
			c.Writer.WriteString(http.StatusText(http.StatusOK))
		}

		c.Next()
	}
}

func main() {
	limiter = zlimiter.NewLimiter(zlimiter.LimitMemFixWindow)
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
