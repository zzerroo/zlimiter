package redis

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/zzerroo/zlimiter/driver/common"
)

const (
	DefaultRedisMaxIdle      = 10
	DefaultRedisMaxActive    = 20
	DefaultRedisIdleTimeout  = 0
	DefaultRedisConnTimeout  = 0
	DefaultRedisReadTimeout  = 0
	DefaultRedisWriteTimeout = 0

	RedisAddScript = "ADD"
	RedisDelScript = "DEL"
	RedisGetScirpt = "GET"
	RedisSetScript = "SET"
)

// RedisInfo ...
type RedisInfo struct {
	Address string
	Passwd  string
}

// RedisProxy ...
type RedisProxy struct {
	//Driver
	RedisClient *redis.Pool
	Scripts     map[int]*redis.Script
}

// RedisFixWindow ...
type RedisFixWindow struct {
	RedisProxy
}

// RedisSlideWindow ...
type RedisSlideWindow struct {
	RedisProxy
}

// RedisBucket ...
type RedisBucket struct {
	RedisProxy
}

// RedisToken ...
type RedisToken struct {
	RedisProxy
}

// Init ...
func (r *RedisProxy) Init(args ...interface{}) error {
	fmt.Printf("%T %v\n", args[0], args[0])
	if len(args) != 1 {
		log.Fatalf("error bad param:%v", args)
	}

	argsTmp := args[0]
	if redisInfo, ok := argsTmp.(RedisInfo); ok {
		r.RedisClient = &redis.Pool{
			MaxIdle:     DefaultRedisMaxIdle,
			MaxActive:   DefaultRedisMaxActive,
			IdleTimeout: time.Duration(DefaultRedisIdleTimeout) * time.Second,
			Wait:        true,
			Dial: func() (redis.Conn, error) {
				con, err := redis.Dial("tcp", redisInfo.Address,
					redis.DialPassword(redisInfo.Passwd),
					redis.DialConnectTimeout(time.Duration(DefaultRedisConnTimeout)*time.Second),
					redis.DialReadTimeout(time.Duration(DefaultRedisReadTimeout)*time.Second),
					redis.DialWriteTimeout(time.Duration(DefaultRedisWriteTimeout)*time.Second))
				if err != nil {
					log.Fatal(err.Error())
				}

				return con, nil
			},
		}

		if erro := r.LoadScript(); erro != nil {
			log.Fatal(erro.Error())
		}

		return nil
	} else {
		log.Fatal(errors.New("error bad param"))
	}

	return nil
}

// LoadScript check wheather the script is loaded, and load the script to redis
func (r *RedisProxy) LoadScript() error {
	conn := r.RedisClient.Get()
	defer conn.Close()

	for key, script := range r.Scripts {
		if loaded, erro := r.scriptLoaded(conn, script.Hash()); erro == nil && loaded == 0 {
			script.Load(conn)
		} else if erro != nil {
			log.Fatalf("error load script %s", key)
		}
	}

	return nil
}

// run SCRIPT EXISTS to check wheather sha1Str is loaded
func (r *RedisProxy) scriptLoaded(conn redis.Conn, sha1Str string) (int, error) {
	rsp, erro := conn.Do("SCRIPT", "EXISTS", sha1Str)
	if erro != nil {
		return -1, erro
	}

	rspArray, ok := rsp.([]interface{})
	if !ok || len(rspArray) != 1 {
		return -1, errors.New("error bad type")
	}

	if isLoaded, ok := rspArray[0].(int64); ok {
		return int(isLoaded), nil
	}

	return -1, errors.New("error bad type")
}

// Init ...
func (r *RedisFixWindow) Init(args ...interface{}) error {
	return r.RedisProxy.Init(args...)
}

// Init ...
func (r *RedisSlideWindow) Init(args ...interface{}) error {
	r.Scripts = make(map[int]*redis.Script)
	r.Scripts[common.RedisAddScript] = redis.NewScript(1, SlideAddStr)
	r.Scripts[common.RedisGetScript] = redis.NewScript(1, SlideGetStr)
	r.Scripts[common.RedisSetScript] = redis.NewScript(1, SlideSetStr)
	r.Scripts[common.RedisDelScript] = redis.NewScript(1, SlideDelStr)

	r.RedisProxy.Init(args...)
	return nil
}

// Init ...
func (r *RedisBucket) Init(args ...interface{}) error {
	r.RedisProxy.Init(args...)
	return nil
}

// Init ...
func (r *RedisToken) Init(args ...interface{}) error {
	r.RedisProxy.Init(args...)
	return nil
}

// Add ...
func (r *RedisProxy) Add(key string, limit int64, tmDuration time.Duration, others ...interface{}) error {
	conn := r.RedisClient.Get()
	defer conn.Close()

	return r.Scripts[common.RedisAddScript].SendHash(conn, key, limit, tmDuration.Nanoseconds()/1e6)
}

// Get ...
func (r *RedisProxy) Get(key string) (bool, int64, error) {
	conn := r.RedisClient.Get()
	defer conn.Close()

	rsp, erro := r.Scripts[common.RedisGetScript].Do(conn, key)
	if erro != nil {
		return false, -1, erro
	}

	if left, ok := rsp.(int64); ok {
		return left < 0, left, nil
	}

	return false, 0, errors.New("error unkown")
}

// Set ...
func (r *RedisProxy) Set(key string, limit int64, tmDuration time.Duration, others ...interface{}) error {
	conn := r.RedisClient.Get()
	defer conn.Close()

	return r.Scripts[common.RedisSetScript].SendHash(conn, key, limit, tmDuration.Nanoseconds()/1e6)
}

// Del ...
func (r *RedisProxy) Del(key string) error {
	conn := r.RedisClient.Get()
	defer conn.Close()

	return r.Scripts[common.RedisDelScript].SendHash(conn, key)
}
