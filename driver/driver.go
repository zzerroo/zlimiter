package driver

import (
	"errors"
	"log"
	"sync"
	"time"
	cache "github.com/zzerroo/zlimiter/driver/memory"
	rds "github.com/zzerroo/zlimiter/driver/redis"

	"github.com/gomodule/redigo/redis"
)

const (
	DEFAULT_PREFIX = "2D1B74349305508B"

	DEFAULT_CACHE_EXPIRE  = 30 * time.Minute
	DEFAULT_PURGES_EXPIRE = 100 * time.Millisecond

	DEFAULT_REDIS_MAX_IDLE       = 10
	DEFAULT_REDIS_MAX_ACTIVE     = 20
	DEFAULT_REDIS_IDLE_TIMEOUT   = 0
	DEFAULT_REDIS_CONN_TIMEOUT   = 0
	DEFAULT_REDIS_READ_TIMEOUT   = 0
	DEFAULT_REDIS_WRIETE_TIMEOUT = 0

	DEFAULT_REDIS_LOCK_KEY = "E57A127B3E895F2B"
	REDIS_ADD_SCRIPT       = "ADD"
	REDIS_DEL_SCRIPT       = "DEL"
	REDIS_GET_SCRIPT       = "GET"
	REDIS_SET_SCRIPT       = "SET"
)

type Driver struct {
	mpData map[string]LimitInfo
	rwMut  sync.RWMutex
}

type LimitInfo struct {
	limits      int64
	tmDuriation time.Duration
}

// RedisInfo ...
type RedisInfo struct {
	Host   string
	Passwd string
}

type DriverI interface {
	Init(...interface{}) error
	Add(string, int64, time.Duration) error
	Get(string) (bool, int64, error)
	Set(string, int64, time.Duration) error
	Delete(string) error
}

type MemDriver struct {
	Driver
	MemCache *cache.Cache
}

type RedisDriver struct {
	Driver
	script      map[string]*redis.Script
	RedisClient *redis.Pool
}

// Init create a buffer cache
func (m *MemDriver) Init(args ...interface{}) error {
	m.MemCache = cache.New(DEFAULT_CACHE_EXPIRE, DEFAULT_PURGES_EXPIRE)
	return nil
}

// Add an item with limit and tmDuriation to the buffer cache
func (m *MemDriver) Add(key string, limits int64, tmDuriation time.Duration) error {
	return m.MemCache.Add(key, limits, tmDuriation)
}

// Get the left times from cache, if left < 0 then return true
func (m *MemDriver) Get(key string) (bool, int64, error) {
	left, erro := m.MemCache.DecrementInt64(key, 1)
	return left < 0, left, erro
}

// Set a new item (limit,tmDuriation) to the cache, a new item will be created
// if item with key is not exist
func (m *MemDriver) Set(key string, limits int64, tmDuriation time.Duration) error {
	m.MemCache.Set(key, limits, tmDuriation)
	return nil
}

// Delete the item
func (m *MemDriver) Delete(key string) error {
	m.MemCache.Delete(key)
	return nil
}

// Init create a redis conn pool with MaxIdle:DEFAULT_REDIS_MAX_IDLE,MaxActive:DEFAULT_REDIS_MAX_ACTIVE,
// it also load the lua script to redis.
func (r *RedisDriver) Init(args ...interface{}) error {
	if len(args) != 1 {
		log.Fatalf("error bad param:%v", args)
	}

	if redisInfo, ok := args[0].(RedisInfo); ok {
		r.RedisClient = &redis.Pool{
			MaxIdle:     DEFAULT_REDIS_MAX_IDLE,
			MaxActive:   DEFAULT_REDIS_MAX_ACTIVE,
			IdleTimeout: time.Duration(DEFAULT_REDIS_IDLE_TIMEOUT) * time.Second,
			Wait:        true,
			Dial: func() (redis.Conn, error) {
				con, err := redis.Dial("tcp", redisInfo.Host,
					redis.DialPassword(redisInfo.Passwd),
					redis.DialConnectTimeout(time.Duration(DEFAULT_REDIS_CONN_TIMEOUT)*time.Second),
					redis.DialReadTimeout(time.Duration(DEFAULT_REDIS_READ_TIMEOUT)*time.Second),
					redis.DialWriteTimeout(time.Duration(DEFAULT_REDIS_WRIETE_TIMEOUT)*time.Second))
				if err != nil {
					log.Fatal(err.Error())
				}

				return con, nil
			},
		}

		r.script = make(map[string]*redis.Script)
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
func (r *RedisDriver) LoadScript() error {
	conn := r.RedisClient.Get()
	defer conn.Close()

	r.script[REDIS_ADD_SCRIPT] = redis.NewScript(1, rds.AddStr)
	r.script[REDIS_GET_SCRIPT] = redis.NewScript(1, rds.GetStr)
	r.script[REDIS_SET_SCRIPT] = redis.NewScript(1, rds.SetStr)
	r.script[REDIS_DEL_SCRIPT] = redis.NewScript(1, rds.DelStr)

	for key, value := range r.script {
		if loaded, erro := r.scriptLoaded(conn, value.Hash()); erro == nil && loaded == 0 {
			value.Load(conn)
		} else if erro != nil {
			log.Fatalf("error load script %s", key)
		}
	}

	return nil
}

// run SCRIPT EXISTS to check wheather sha1Str is loaded
func (r *RedisDriver) scriptLoaded(conn redis.Conn, sha1Str string) (int, error) {
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

// Add a limit item with (key,limit,duration)
func (r *RedisDriver) Add(key string, limit int64, tmDuration time.Duration) error {
	conn := r.RedisClient.Get()
	defer conn.Close()

	return r.script[REDIS_ADD_SCRIPT].SendHash(conn, key, limit, tmDuration.Nanoseconds()/1e6)
}

// Get the limit info from redis. if there is no times left(left < 0),true will be returned
func (r *RedisDriver) Get(key string) (bool, int64, error) {
	conn := r.RedisClient.Get()
	defer conn.Close()

	rsp, erro := r.script[REDIS_GET_SCRIPT].Do(conn, key)
	if erro != nil {
		return false, -1, erro
	}

	if left, ok := rsp.(int64); ok {
		return left < 0, left, nil
	}

	return false, 0, errors.New("error unkown")
}

// Set update or insert a new item,this will update the limit counter
func (r *RedisDriver) Set(key string, limit int64, tmDuration time.Duration) error {
	conn := r.RedisClient.Get()
	defer conn.Close()

	return r.script[REDIS_SET_SCRIPT].SendHash(conn, key, limit, tmDuration.Nanoseconds()/1e6)
}

// Delete the limit info from redis
func (r *RedisDriver) Delete(key string) error {
	conn := r.RedisClient.Get()
	defer conn.Close()

	return r.script[REDIS_DEL_SCRIPT].SendHash(conn, key)
}
