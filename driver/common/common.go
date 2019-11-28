package common

const (
	LimitMemFixWindow     = 201
	LimitMemSlideWindow   = 202
	LimitMemBucket        = 203
	LimitMemToken         = 204
	LimitRedisFixWindow   = 205
	LimitRedisSlideWindow = 206
	LimitRedisBucket      = 207
	LimitRedisToken       = 208

	RedisAddScript = 301
	RedisGetScript = 302
	RedisSetScript = 303
	RedisDelScript = 304
	ReidsChkScript = 305

	ErrorUnknown      = "error unkown error"
	ErrorInputParam   = "error bad input param"
	ErrorReqOverFlow  = "error request overflow"
	ErrorItemNotExist = "error item not exist"
	ErrorLoadScritp   = "error load script"

	ErrorReturnNoLeft       = -1
	ErrorReturnItemNotExist = -1001
	ErrorReturnNoMeans      = -1002
	ErrorReturnBucket       = -1003
)

// // Limiter ...
// type Limiter interface {
// 	Init() error
// 	Add(string, int64, time.Duration, ...interface{}) error
// 	Set(string, int64, time.Duration, ...interface{}) error
// 	Del(string) error
// 	Get(string) (bool, int64, error)
// }
