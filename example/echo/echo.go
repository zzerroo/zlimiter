package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/zzerroo/zlimiter"

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

		left, erro := l.L.Get(ips[0])
		if left == zlimiter.ErrorReturnItemNotExist { // nerver add
			l.L.Add(ips[0], 10, 1*time.Second)
			l.L.Get(ips[0])
		}

		if left >= 0 && erro != nil {
			return c.String(http.StatusOK, http.StatusText(http.StatusOK))
		}

		if left == -1 { // reach the limit
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

	l.L = zlimiter.NewLimiter(zlimiter.LimitMemFixWindow)

	e.Logger.Fatal(e.Start(":8080"))
}
