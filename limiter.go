package zlimiter

import (
	"time"
	"github.com/zzerroo/zlimiter/driver"
)

const (
	LIMIT_TYPE_MEM   = 0
	LIMIT_TYPE_REDIS = 1
)

// Limits ...
type Limits struct {
	Driver driver.DriverI
	left   int64
}

// NewLimiter create a limiter with cacheType、args and init the buffer(or conn pool)
// LIMIT_TYPE_MEM create a limiter based on buffer cache, LIMIT_TYPE_REDIS create a
// dist limiter based on redis. if the limiter type is LIMIT_TYPE_REDIS, args include
// the redis info
func NewLimiter(cacheType int64, args ...interface{}) (*Limits, error) {
	limiter := new(Limits)
	if cacheType == LIMIT_TYPE_MEM {
		limiter.Driver = &driver.MemDriver{}
		limiter.Driver.Init(nil)
	} else if cacheType == LIMIT_TYPE_REDIS {
		limiter.Driver = &driver.RedisDriver{}
		limiter.Driver.Init(args...)
	}

	return limiter, nil
}

// Add a limit item to local buffer or redis, limit is the limit count for the key
// tmSpan is the duration. The redis-based limiter can only be accurate to milliseconds.
func (l *Limits) Add(key string, limit int64, tmSpan time.Duration) error {
	return l.Driver.Add(key, limit, tmSpan)
}

// Set update or insert limit item,with the limit info {limits,tmDuration}
func (l *Limits) Set(key string, limits int64, tmDuration time.Duration) error {
	return l.Driver.Set(key, limits, tmDuration)
}

// Get the left count of the key
func (l *Limits) Get(key string) (bool, int64, error) {
	reached, left, erro := l.Driver.Get(key)
	return reached, left, erro
}

// Delete key from buffer of redis
func (l *Limits) Delete(key string) error {
	return l.Driver.Delete(key)
}
