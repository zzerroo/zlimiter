package zlimiter_test

import (
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/zzerroo/zlimiter"
	rds "github.com/zzerroo/zlimiter/driver/redis"
)

func TestRedisFixWindow(t *testing.T) {

	key := "test"
	redisLimit := zlimiter.NewLimiter(zlimiter.LimitRedisFixWindow, rds.RedisInfo{Address: "127.0.0.1:6379", Passwd: "test"})

	// Test Add
	erro := redisLimit.Add(key, 10, 2*time.Second)
	if erro != nil {
		t.Error(erro.Error())
	}

	// Test Get
	left, erro := redisLimit.Get(key)
	if left != 9 || erro != nil {
		t.Errorf("%v %v, should be 9,nil", left, erro)
	}

	// Test timeout
	time.Sleep(3 * time.Second)
	left, erro = redisLimit.Get(key)
	if left != 9 || erro != nil {
		t.Errorf("%v %v,should be 9,nil", left, erro)
	}

	// Test Set
	erro = redisLimit.Set(key, 15, 4*time.Second)
	if erro != nil {
		t.Error(erro.Error())
	}

	left, erro = redisLimit.Get(key)
	if left != 14 || erro != nil {
		t.Errorf("%v %v,should be 14,nil", left, erro)
	}

	// Test Sync Get
	var successCnt, failCnt int
	var wg sync.WaitGroup
	for i := 0; i < 18; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			left, erro := redisLimit.Get(key)
			if left == -1 || erro != nil {
				failCnt++
			} else {
				successCnt++
			}
		}(i)
	}

	wg.Wait()
	if failCnt != 4 {
		t.Error(failCnt)
	}

	key = "test1"
	left, erro = redisLimit.Get(key)
	if left != zlimiter.ErrorReturnItemNotExist || erro != nil {
		t.Errorf("%v,%v,should -1001,not nil", left, erro)
	}
	key = "test"

	// Test Del
	erro = redisLimit.Del(key)
	if erro != nil {
		t.Error(erro.Error())
	}

	left, _ = redisLimit.Get(key)
	if left != zlimiter.ErrorReturnItemNotExist {
		t.Error("item should not exist")
	}
}

func TestRedisSlideWindow(t *testing.T) {

	key := "test"
	redisLimit := zlimiter.NewLimiter(zlimiter.LimitRedisSlideWindow, rds.RedisInfo{Address: "127.0.0.1:6379", Passwd: "test"})

	// Test Add
	erro := redisLimit.Add(key, 10, 2*time.Second)
	if erro != nil {
		t.Error(erro.Error())
	}

	// Test Get
	left, erro := redisLimit.Get(key)
	if left != 9 || erro != nil {
		t.Errorf("%v %v,should be 9 nil", left, erro)
	}

	// Test timeout
	time.Sleep(3 * time.Second)
	left, erro = redisLimit.Get(key)
	if left != 9 || erro != nil {
		t.Errorf("%v %v,should be 9 nil", left, erro)
	}

	// Test Set
	erro = redisLimit.Set(key, 15, 4*time.Second)
	if erro != nil {
		t.Error(erro.Error())
	}

	left, erro = redisLimit.Get(key)
	if left != 14 || erro != nil {
		t.Errorf("%v %v,should be 14 nil", left, erro)
	}

	// Test Sync Get
	var successCnt, failCnt int
	var wg sync.WaitGroup
	for i := 0; i < 18; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			left, erro := redisLimit.Get(key)
			if left == -1 || erro != nil {
				failCnt++
			} else {
				successCnt++
			}
		}()
	}

	wg.Wait()
	if failCnt != 4 {
		t.Errorf("%v should be 4", failCnt)
	}

	key = "test1"
	left, erro = redisLimit.Get(key)
	if left != zlimiter.ErrorReturnItemNotExist || erro != nil {
		t.Errorf("%v,%vshould -1001,not nil", left, erro)
	}
	key = "test"

	// test overflow
	erro = redisLimit.Set(key, 15, 4*time.Second)
	if erro != nil {
		t.Error(erro.Error())
	}

	time.Sleep(1 * time.Second)
	left, erro = redisLimit.Get(key)
	if left != 14 || erro != nil {
		t.Errorf("%v %v,should be 14 nil", left, erro)
	}

	time.Sleep(1 * time.Second)
	left, erro = redisLimit.Get(key)
	if left != 13 || erro != nil {
		t.Errorf("%v %v,should be 13 nil", left, erro)
	}

	time.Sleep(1 * time.Second)
	left, erro = redisLimit.Get(key)
	if left != 12 || erro != nil {
		t.Errorf("%v %v,should be 12 nil", left, erro)
	}

	time.Sleep(2 * time.Second)
	left, erro = redisLimit.Get(key)
	if left != 12 || erro != nil {
		t.Errorf("%v %v,should be 12 nil", left, erro)
	}

	// Test Del
	erro = redisLimit.Del(key)
	if erro != nil {
		t.Error(erro.Error())
	}

	left, _ = redisLimit.Get(key)
	if left != zlimiter.ErrorReturnItemNotExist {
		t.Errorf("%v,item should be not exist", erro)
	}
}

func TestRedisToken(t *testing.T) {
	key := "test"
	var left, max int64 = 0, 20

	// create
	redisLimit := zlimiter.NewLimiter(zlimiter.LimitRedisToken, rds.RedisInfo{Address: "127.0.0.1:6379", Passwd: "test"})

	// test add
	erro := redisLimit.Add(key, 4, 4*time.Second, max)
	if erro != nil {
		t.Errorf("error:%s", erro.Error())
	}

	time.Sleep(1 * time.Second)

	// reached,left == false,0
	left, erro = redisLimit.Get(key)
	if left != 0 || erro != nil {
		t.Errorf("%v,%v,should be 0, nil", left, erro)
	}
	// reached,left == true,-1
	left, erro = redisLimit.Get(key)
	if left != zlimiter.ErrorReturnNoLeft || erro != nil {
		t.Errorf("%v,%v,should be -1, nil", left, erro)
	}

	// create 4 token
	time.Sleep(4 * time.Second)

	// test get and limit
	sCnt := 0
	fCnt := 0
	var wg sync.WaitGroup
	for i := 0; i < 14; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			left, erro := redisLimit.Get(key)
			if erro == nil && left >= 0 {
				sCnt++
			} else {
				fCnt++
			}
		}()
	}

	wg.Wait()

	//sCnt == 4
	if sCnt != 4 {
		t.Errorf("sCnt %d,should be 4", sCnt)
	}

	key = "test1"
	left, erro = redisLimit.Get(key)
	if left != zlimiter.ErrorReturnItemNotExist || erro != nil {
		t.Errorf("%v,%v,should -1001,not nil", left, erro)
	}
	key = "test"

	// test set
	erro = redisLimit.Set(key, 4, 2*time.Second, max)
	if erro != nil {
		t.Errorf("error %s", erro.Error())
	}

	time.Sleep(4 * time.Second)

	// reached,left == false,7
	left, erro = redisLimit.Get(key)
	if left != 7 {
		t.Errorf("%v,%v,should be 7, nil", left, erro)
	}

	// test overflow
	erro = redisLimit.Set(key, 4, 4*time.Second, max)
	if erro != nil {
		t.Errorf("error %s", erro.Error())
	}

	time.Sleep(1 * time.Second)

	// reached,left == false 0
	left, erro = redisLimit.Get(key)
	if left != 0 {
		t.Errorf("%v,%v,should be 0, nil", left, erro)
	}

	time.Sleep(25 * time.Second)

	// reached,left == false 19
	left, erro = redisLimit.Get(key)
	if left != 19 {
		t.Errorf("%v,%v,should be 19, nil", left, erro)
	}

	// reached,left == false 18
	left, erro = redisLimit.Get(key)
	if left != 18 {
		t.Errorf("%v,%v,should be 18, nil", left, erro)
	}

	// test del
	redisLimit.Del(key)
	left, _ = redisLimit.Get(key)
	if left != zlimiter.ErrorReturnItemNotExist {
		t.Error("should not find the key")
	}
}

func TestRedisBucket(t *testing.T) {
	// test add
	key := "test"
	var left, max, sCnt, fCnt int64 = 0, 20, 0, 0

	redisLimit := zlimiter.NewLimiter(zlimiter.LimitRedisBucket, rds.RedisInfo{Address: "127.0.0.1:6379", Passwd: "test"})

	erro := redisLimit.Add(key, 4, 4*time.Second, max)
	if erro != nil {
		t.Errorf("error:%s", erro.Error())
	}

	// reached，left == false,-1
	tm1 := time.Now()
	left, erro = redisLimit.Get(key)
	if erro != nil || left != zlimiter.ErrorReturnBucket {
		t.Errorf("%v,%v,should -1003,nil", left, erro)
	}

	// reached,left == true,0
	left, erro = redisLimit.Get(key)
	if erro != nil || left != zlimiter.ErrorReturnBucket {
		t.Errorf("%v,%vshould -1003,nil", left, erro)
	}
	tm2 := time.Now()

	// duration about 1s
	duSec := tm2.Sub(tm1).Seconds()
	if int64(duSec) != 1 {
		t.Errorf("%v,tm duration should be 1 sec", int64(duSec))
	}

	// test get and limit
	var wg sync.WaitGroup
	for i := 0; i < 14; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			left, erro := redisLimit.Get(key)
			if erro == nil && left == zlimiter.ErrorReturnBucket {
				sCnt++
			} else {
				fCnt++
			}
		}()
	}

	wg.Wait()
	if sCnt != 14 {
		t.Errorf("%v,sCnt should be 13", sCnt)
	}

	// test key not exist
	key = "test1"
	left, erro = redisLimit.Get(key)
	if left != zlimiter.ErrorReturnItemNotExist || erro != nil {
		t.Errorf("%v,%vshould -1001,not nil", left, erro)
	}
	key = "test"

	// test set
	erro = redisLimit.Set(key, 4, 8*time.Second, max)
	if erro != nil {
		t.Errorf("error %s", erro.Error())
	}

	// reached,left == false,-1
	tm1 = time.Now()
	left, erro = redisLimit.Get(key)
	if left != zlimiter.ErrorReturnBucket || erro != nil {
		t.Errorf("%v,%v should -1003,nil", left, erro)
	}

	// reached,left == false,-1
	left, erro = redisLimit.Get(key)
	if left != zlimiter.ErrorReturnBucket || erro != nil {
		t.Errorf("%v,%v should -1003,nil", left, erro)
	}
	tm2 = time.Now()

	duSec = tm2.Sub(tm1).Seconds()
	if int64(duSec) != 2 {
		t.Errorf("%v, int64(duSec) should be 2 sec", int64(duSec))
	}

	erro = redisLimit.Set(key, 4, 8*time.Second, max)
	if erro != nil {
		t.Errorf("error %s", erro.Error())
	}

	//reached,left == false 0
	tm1 = time.Now()
	left, erro = redisLimit.Get(key)
	if left != zlimiter.ErrorReturnBucket || erro != nil {
		t.Errorf("%v,%v,should -1003,nil", left, erro)
	}

	time.Sleep(1500 * time.Millisecond)

	// reached,left == false,0
	left, erro = redisLimit.Get(key)
	if left != zlimiter.ErrorReturnBucket || erro != nil {
		t.Errorf("%v,%v should 1003,nil", left, erro)
	}

	tm2 = time.Now()
	duMs := tm2.Sub(tm1).Nanoseconds() / 1e6
	if math.Abs(float64(int64(duMs)-2000)) >= 100 {
		t.Errorf("%v,math.Abs(float64(int64(duMs)-2000)) >= 100", math.Abs(float64(int64(duMs)-2000)))
	}

	// test del
	redisLimit.Del(key)
	left, _ = redisLimit.Get(key)
	if left != zlimiter.ErrorReturnItemNotExist {
		t.Error("should not find the key")
	}
}

func TestRealRedis(t *testing.T) {

	var left, sucCnt, failCnt int64
	var erro error
	ipLocal := getLocalIP()

	var windowTypes = [4]int64{zlimiter.LimitRedisFixWindow, zlimiter.LimitRedisSlideWindow, zlimiter.LimitRedisBucket, zlimiter.LimitRedisToken}

	for _, wdwType := range windowTypes {
		redisLimit := zlimiter.NewLimiter(int64(wdwType), rds.RedisInfo{Address: "127.0.0.1:6379", Passwd: "test"})
		sucCnt = 0
		failCnt = 0

		left, erro = redisLimit.Get(ipLocal)
		if erro == nil && left == zlimiter.ErrorReturnItemNotExist {
			redisLimit.Add(ipLocal, 10, 1*time.Second, 30)
		}

		if wdwType == zlimiter.LimitRedisToken {
			time.Sleep(1000 * time.Millisecond)
		}

		var wg sync.WaitGroup
		for i := 0; i < 13; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				left, erro = redisLimit.Get(ipLocal)
				if erro != nil {
					fmt.Println(erro)
					failCnt++
				} else if left >= 0 || (left == zlimiter.ErrorReturnBucket && wdwType == zlimiter.LimitRedisBucket) {
					sucCnt++
				} else if left < 0 {
					failCnt++
				}
			}(i)
		}

		wg.Wait()
		if wdwType == zlimiter.LimitRedisFixWindow {
			if sucCnt != 10 || failCnt != 3 {
				t.Errorf("LimitRedisFixWindow sucCnt != 10 && failCnt != 13,%v %v", sucCnt, failCnt)
			}
		} else if wdwType == zlimiter.LimitRedisSlideWindow {
			if sucCnt != 10 || failCnt != 3 {
				t.Errorf("LimitRedisSlideWindow sucCnt != 10 && failCnt != 13,%v %v", sucCnt, failCnt)
			}
		} else if wdwType == zlimiter.LimitRedisBucket {
			if sucCnt != 13 {
				t.Errorf("LimitRedisBucket sucCnt != 13,%v", sucCnt)
			}
		} else if wdwType == zlimiter.LimitRedisToken {
			if sucCnt != 10 || failCnt != 3 {
				t.Errorf("LimitRedisToken sucCnt != 10 && failCnt != 13,%v %v", sucCnt, failCnt)
			}
		}

		redisLimit.Del(ipLocal)
	}
}
