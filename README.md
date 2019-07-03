# zlimiter

zlimiter is a k/v based rate limite library, support counter algorithm  based on  memory or redis, and it is thread safe.[Chinese Version](./readme-ch.md)

# Usage

## Install

glide get github.com/zzerroo/zlimiter

## Usage

memory：

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

