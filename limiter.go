package zlimiter

import (
	"log"
	"time"

	"github.com/zzerroo/zlimiter/driver/common"
	memory "github.com/zzerroo/zlimiter/driver/memory"
	rds "github.com/zzerroo/zlimiter/driver/redis"
)

const (
	// LimitMemFixWindow 用于标识基于内存的固定窗口限流
	LimitMemFixWindow = common.LimitMemFixWindow
	// LimitMemSlideWindow 用于标识基于内存的滑动窗口限流
	LimitMemSlideWindow = common.LimitMemSlideWindow
	// LimitMemBucket 用于标识基于内存的bucket限流
	LimitMemBucket = common.LimitMemBucket
	// LimitMemToken 用于标识基于内存的Token限流
	LimitMemToken = common.LimitMemToken
	// LimitRedisFixWindow 用于标识分布式固定窗口限流
	LimitRedisFixWindow = common.LimitRedisFixWindow
	// LimitRedisSlideWindow 用于标识分布式滑动窗口限流
	LimitRedisSlideWindow = common.LimitRedisSlideWindow
	// LimitRedisBucket 用于标识分布式bucket限流
	LimitRedisBucket = common.LimitRedisBucket
	// LimitRedisToken 用于标识分布式token限流
	LimitRedisToken = common.LimitRedisToken
)

// DriverI zlimiter驱动接口，任何zlimiter的driver都要实现该接口
// Add、Get、Set、Del分别为新增、获取、设置、删除的相关处理函数
type DriverI interface {
	Init(...interface{}) error
	Add(string, int64, time.Duration, ...interface{}) error
	Get(string) (bool, int64, error)
	Set(string, int64, time.Duration, ...interface{}) error
	Del(string) error
}

// Limits zlimiter实例类，所有功能都是通过该struct实现
type Limits struct {
	driver DriverI
}

// NewLimiter 工厂函数，用于创建一个限流器，关于各种限流器的说明请参见：,如果限流器创建过程中
//	发生错误，则会Log.Fatal错误信息并停止，否则返回相关限流器
// Input:
//	limiterType : 需要创建的限流器的类型,具体包括：
//		LimitMemFixWindow	单机固定窗口限流
//		LimitMemSlideWindow	单机滑动窗口限流
//		LimitMemToken	单机token限流
//		LimitMemBucket	单机桶限流
//		LimitRedisFixWindow 分布式固定窗口限流
//		LimitRedisSlideWindow	分布式滑动窗口限流
//		LimitRedisBucket	分布式桶限流
//		LimitRedisToken	分布式token限流
//	args : 初始化参数，主要用于Redis相关的初始化，注意redis初始化需要RedisInfo类型的变量
// Output:
//	*Limits : 成功创建的限流器
func NewLimiter(limiterType int64, args ...interface{}) *Limits {
	limiter := new(Limits)

	if limiterType == common.LimitMemFixWindow {
		limiter.driver = new(memory.CacheFixWindow)
	} else if limiterType == common.LimitMemSlideWindow {
		limiter.driver = new(memory.CacheSlideWindow)
	} else if limiterType == common.LimitMemBucket {
		limiter.driver = new(memory.Bucket)
	} else if limiterType == common.LimitMemToken {
		limiter.driver = new(memory.Token)
	} else if limiterType == common.LimitRedisFixWindow {
		limiter.driver = new(rds.RedisFixWindow)
	} else if limiterType == common.LimitRedisSlideWindow {
		limiter.driver = new(rds.RedisSlideWindow)
	} else if limiterType == common.LimitRedisBucket {
		limiter.driver = new(rds.RedisBucket)
	} else if limiterType == common.LimitRedisToken {
		limiter.driver = new(rds.RedisToken)
	} else {
		log.Fatalf(common.ErrorInputParam)
	}

	limiter.driver.Init(args...)
	return limiter
}

// Add 创建一条基于key的限流规则
//	Input :
//		key : 限流标识，用于唯一标识一条限流规则
//		limit : tmSpan时间段内的限流数，与tmSpan同时实现tmSpan时间段内限流limit次语义
//		tmSpan : 时间段,与limit同时实现tmSpan时间段内限流limit次语义
//		others : 其他相关参数，目前bucket和token限流的max值
//	Output :
//		error : 成功为nil，否则为具体错误信息
func (l *Limits) Add(key string, limit int64, tmSpan time.Duration, others ...interface{}) error {
	return l.driver.Add(key, limit, tmSpan, others...)
}

// Set 创建或者重置基于key的限流规则
//	Input :
//		key : 限流标识，用于唯一标识一条限流规则
//		limit : tmSpan时间段内的限流数，与tmSpan同时实现tmSpan时间段内限流limit次语义
//		tmSpan : 时间段,与limit同时实现tmSpan时间段内限流limit次语义
//		others : 其他相关参数，目前bucket和token限流的max值
//	Output :
//		error : 成功为nil，否则为具体错误信息
func (l *Limits) Set(key string, limits int64, tmDuration time.Duration, others ...interface{}) error {
	return l.driver.Set(key, limits, tmDuration, others...)
}

// Get 获取基于key的请求的相关信息，包括：本次访问是否可以放行，剩余可放行的访问数
//	Input :
//		key : 限流标识，对应于限流规则
//	Output :
//		bool : 访问是否可以放行，当剩余可访问次数<0时候，为true，否则为false。注意，基于bucket的访问
//			本参数永远为false
//		int64 : 剩余的可访问次数, 注意，基于bucket本参数永远为-1
//		error : 相关错误信息，成功为nil 否则为相关错误
func (l *Limits) Get(key string) (bool, int64, error) {
	reached, left, erro := l.driver.Get(key)
	return reached, left, erro
}

// Del 删除基于key的限流
//	Input :
//		key : 要删除的限流的标识
//	Output :
//		error : 相关错误信息，成功为nil 否则为相关错误
func (l *Limits) Del(key string) error {
	return l.driver.Del(key)
}
