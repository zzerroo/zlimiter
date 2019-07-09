# zlimiter

zlimiter is a k/v based rate limite library, support counter algorithm  based on  memory or redis, and it is thread safe.[Chinese Version](./readme-ch.md)

The implementation of the LIMIT_TYPE_MEM mode is based on go-cache (https://github.com/patrickmn/go-cache), which is a K/V based cache system with expiration mechanism, thread safety and so on, Mechanisms such as expiration are also based on go-cache. 

LIMIT_TYPE_REDIS mode is based on the redis for thread safe and expiration .In order to improve the efficiency and ensure the atomicity of the operation, this mode use lua script to implement the create, delete and modify operations (see lua-string.go for details). For the operation of redis, the redigo pool  is used. If you need to adjust the redis configuration, see driver.go.

# Usage

## Install

glide get github.com/zzerroo/zlimiter

## Usage

local memory：

```go
 key := "test" 
 // create
	memLimit, erro := zlimiter.NewLimiter(zlimiter.LIMIT_TYPE_MEM)
	if erro != nil {
		log.Errorf("error:%s", erro.Error())
	}

	// Add
	key := "test"
	erro = memLimit.Add(key, 3, 4*time.Second)
	if erro != nil {
		log.Errorf("error:%s", erro.Error())
	}

	// Set
	erro = memLimit.Set(key, 100, 10*time.Second)
	if erro != nil {
		log.Errorf("error %s", erro.Error())
	}

	// Get
	reach, left, erro := memLimit.Get(key)
	if erro != nil || reach != false || left != 99 {
		log.Errorf("reach %v,left %d, error %s", reach, left, erro.Error())
	}

	// Delete
	memLimit.Delete(key)
```



redis：

```go
	key := "test"
	// create
	redisLimit, erro := zlimiter.NewLimiter(zlimiter.LIMIT_TYPE_REDIS, driver.RedisInfo{Host: "127.0.0.1:6379", Passwd: "passwd"})
	if erro != nil {
		log.Error(erro.Error())
	}

	// Add
	erro = redisLimit.Add(key, 10, 2*time.Second)
	if erro != nil {
		log.Error(erro.Error())
	}

	// Get
	bReached, left, erro := redisLimit.Get(key)
	if bReached != false || left != 9 || erro != nil {
		log.Error(bReached, left, erro)
	}

	// Set
	erro = redisLimit.Set(key, 15, 50*time.Second)
	if erro != nil {
		log.Error(erro.Error())
	}

	// Del
	redisLimit.Delete(key)
```



- [beego](./example/beego/beego.go)
- [echo](./example/echo/echo.go)
- [gin](./example/gin/gin.go)
- [http](./example/http/http.go)
