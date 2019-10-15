package zlimiter_test

import (
	"sync"
	"testing"
	"time"
	"github.com/zzerroo/zlimiter"
	"github.com/zzerroo/zlimiter/driver"
)

func TestAll(t *testing.T) {

	key := "test"
	redisLimit, erro := zlimiter.NewLimiter(zlimiter.LIMIT_TYPE_REDIS, driver.RedisInfo{Host: "10.96.81.176:6379", Passwd: "my_redis"})
	if erro != nil {
		t.Error(erro.Error())
	}

	// Test Add
	erro = redisLimit.Add(key, 10, 2*time.Second)
	if erro != nil {
		t.Error(erro.Error())
	}

	// Test Get
	bReached, left, erro := redisLimit.Get(key)
	if bReached != false || left != 9 || erro != nil {
		t.Error(bReached, left, erro)
	}

	// Test timeout
	time.Sleep(3 * time.Second)
	bReached, left, erro = redisLimit.Get(key)
	if bReached != false || left != 9 || erro != nil {
		t.Error(bReached, left, erro)
	}

	// Test Set
	erro = redisLimit.Set(key, 15, 50*time.Second)
	if erro != nil {
		t.Error(erro.Error())
	}

	bReached, left, erro = redisLimit.Get(key)
	if bReached != false || left != 14 || erro != nil {
		t.Error(bReached, left, erro)
	}

	var successCnt, failCnt int
	var wg sync.WaitGroup
	// Test Get
	for i := 0; i < 18; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			bReached, left, erro := redisLimit.Get(key)
			if bReached != false || left < 0 || erro != nil {
				failCnt++
			} else {
				successCnt++
			}

			// // reach the limit,bReached should be true and left should be -1
			// if 13-idx < 0 && (bReached != false || left != -1) {
			// 	failCnt++
			// }
		}(i)
	}

	wg.Wait()
	if failCnt != 4 {
		t.Error(failCnt)
	}

	// Test Del
	erro = redisLimit.Delete(key)
	if erro != nil {
		t.Error(erro.Error())
	}

	bReached, left, erro = redisLimit.Get(key)
	if bReached != true || left != -2 || erro != nil {
		t.Error(bReached, left, erro)
	}
}
