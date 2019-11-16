package zlimiter

import (
	"log"
	"time"

	"github.com/zzerroo/zlimiter/driver/common"
	memory "github.com/zzerroo/zlimiter/driver/memory"
	rds "github.com/zzerroo/zlimiter/driver/redis"
)

const (
	LimitMemFixWindow     = common.LimitMemFixWindow
	LimitMemSlideWindow   = common.LimitMemSlideWindow
	LimitMemBucket        = common.LimitMemBucket
	LimitMemToken         = common.LimitMemToken
	LimitRedisFixWindow   = common.LimitRedisFixWindow
	LimitRedisSlideWindow = common.LimitRedisSlideWindow
	LimitRedisBucket      = common.LimitRedisBucket
	LimitRedisToken       = common.LimitMemToken
)

// DriverI ...
type DriverI interface {
	Init(...interface{}) error
	Add(string, int64, time.Duration, ...interface{}) error
	Get(string) (bool, int64, error)
	Set(string, int64, time.Duration, ...interface{}) error
	Del(string) error
}

// Limits ...
type Limits struct {
	Driver DriverI
}

// NewLimiter create a limiter with cacheType、args and init the buffer(or conn pool)
// LIMIT_TYPE_MEM create a limiter based on buffer cache, LIMIT_TYPE_REDIS create a
// dist limiter based on redis. if the limiter type is LIMIT_TYPE_REDIS, args include
// the redis info
func NewLimiter(limiterType int64, args ...interface{}) (*Limits, error) {
	limiter := new(Limits)

	if limiterType == common.LimitMemFixWindow {
		limiter.Driver = new(memory.CacheFixWindow)
	} else if limiterType == common.LimitMemSlideWindow {
		limiter.Driver = new(memory.CacheSlideWindow)
	} else if limiterType == common.LimitMemBucket {
		limiter.Driver = new(memory.Bucket)
	} else if limiterType == common.LimitMemToken {
		limiter.Driver = new(memory.Token)
	} else if limiterType == common.LimitRedisFixWindow {
		limiter.Driver = new(rds.RedisFixWindow)
	} else if limiterType == common.LimitRedisSlideWindow {
		limiter.Driver = new(rds.RedisSlideWindow)
	} else if limiterType == common.LimitRedisBucket {
		limiter.Driver = new(rds.RedisBucket)
	} else if limiterType == common.LimitRedisToken {
		limiter.Driver = new(rds.RedisToken)
	} else {
		log.Fatalf(common.ErrorInputParam)
	}

	limiter.Driver.Init(args...)
	return limiter, nil
}

// Add a limit item to local buffer or redis, limit is the limit count for the key
// tmSpan is the duration. The redis-based limiter can only be accurate to milliseconds.
func (l *Limits) Add(key string, limit int64, tmSpan time.Duration, others ...interface{}) error {
	return l.Driver.Add(key, limit, tmSpan, others...)
}

// Set update or insert limit item,with the limit info {limits,tmDuration}
func (l *Limits) Set(key string, limits int64, tmDuration time.Duration, others ...interface{}) error {
	return l.Driver.Set(key, limits, tmDuration, others...)
}

// Get the left count of the key
func (l *Limits) Get(key string) (bool, int64, error) {
	reached, left, erro := l.Driver.Get(key)
	return reached, left, erro
}

// Del key from buffer of redis
func (l *Limits) Del(key string) error {
	return l.Driver.Del(key)
}
