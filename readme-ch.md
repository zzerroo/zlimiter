# zlimiter
zlimiter是线程安全的基于k/v的限流系统，用于系统的限流。支持单机(基于内存,LIMIT_TYPE_MEM)、分布式(基于redis,LIMIT_TYPE_REDIS)的限流方式，目前仅支持计数方式的限流。[英文版](./readme.md)

LIMIT_TYPE_MEM模式的实现基于了go-cache，该库是一个具有过期机制、线程安全的K/V缓存系统，LIMIT_TYPE_MEM模式的原子操作、计数、过期等机制是也是基于go-cache的。此外go-cache通过读写锁保证同步操作。

LIMIT_TYPE_REDIS模式利用了redis的线程安全、过期等机制，为了提升效率和保证操作的原子性，该模式利用了lua脚本实现了相关新增、删除、修改机制（具体参见lua-string.go）。对于redis的操作采用了redigo中池相关机制，如需调整相关配置，请修改DEFAULT_REDIS_*，位于driver.go。

# Usage

## Install

glide get github.com/zzerroo/zlimiter

## Usage

基于内存：

```go
 key := "test" 
 // create
	memLimit, erro := zlimiter.NewLimiter(zlimiter.LIMIT_TYPE_MEM)
	if erro != nil {
		log.Fatalf("error:%s", erro.Error())
	}

	// Add
	key := "test"
	erro = memLimit.Add(key, 3, 4*time.Second)
	if erro != nil {
		log.Fatalf("error:%s", erro.Error())
	}

	// Set
	erro = memLimit.Set(key, 100, 10*time.Second)
	if erro != nil {
		log.Fatalf("error %s", erro.Error())
	}

	// Get
	reach, left, erro := memLimit.Get(key)
	if erro != nil || reach != false || left != 99 {
		log.Fatalf("reach %v,left %d, error %s", reach, left, erro.Error())
	}

	// Delete
	memLimit.Delete(key)
```



基于Redis：

```go
	key := "test"
	// create
	redisLimit, erro := zlimiter.NewLimiter(zlimiter.LIMIT_TYPE_REDIS, driver.RedisInfo{Host: "127.0.0.1:6379", Passwd: "passwd"})
	if erro != nil {
		log.Fatal(erro.Error())
	}

	// Add
	erro = redisLimit.Add(key, 10, 2*time.Second)
	if erro != nil {
		log.Fatal(erro.Error())
	}

	// Get
	bReached, left, erro := redisLimit.Get(key)
	if bReached != false || left != 9 || erro != nil {
		log.Fatal(bReached, left, erro)
	}

	// Set
	erro = redisLimit.Set(key, 15, 50*time.Second)
	if erro != nil {
		log.Fatal(erro.Error())
	}

	// Del
	redisLimit.Delete(key)
```



- [beego中使用](./example/beego/beego.go)
- [echo中使用](./example/echo/echo.go)
- [gin中使用](./example/gin/gin.go)
- [http使用](./example/http/http.go)



