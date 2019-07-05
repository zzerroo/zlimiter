package main

import (
	"log"
	"net/http"
	"strings"
	"time"
	"zlimiter"
	"zlimiter/driver"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

type Limiter struct {
	L *zlimiter.Limits
}

// Limit ...
func (l *Limiter) Limit(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		//c.RealIP()
		ipInfos := c.Request().RemoteAddr
		ips := strings.Split(ipInfos, ":")

		bReached, left, erro := l.L.Get(ips[0])
		if erro != nil {
			log.Fatal(erro.Error())
		}
		if left == -2 { // nerver add
			l.L.Add(ips[0], 10, 2*time.Second)
			l.L.Get(ips[0])
		}

		if bReached == true && left == -1 { // reach the limit
			return c.String(http.StatusTooManyRequests, http.StatusText(http.StatusTooManyRequests))
		}

		return next(c)
	}
}

var erro error

func main() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	l := &Limiter{L: nil}
	e.Use(l.Limit)

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, http.StatusText(http.StatusOK))
	})

	l.L, erro = zlimiter.NewLimiter(zlimiter.LIMIT_TYPE_REDIS, driver.RedisInfo{Host: "127.0.0.1:6379", Passwd: "xxxx"})

	e.Logger.Fatal(e.Start(":8080"))
}
