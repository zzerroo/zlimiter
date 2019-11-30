package zlimiter_test

import (
	"fmt"
	"math"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/zzerroo/zlimiter"
)

func TestMem(t *testing.T) {
	var windowTypes = [2]int64{zlimiter.LimitMemFixWindow, zlimiter.LimitMemSlideWindow}
	for _, wdwType := range windowTypes {
		memLimit := zlimiter.NewLimiter(int64(wdwType))

		// test add
		key := "test"
		var left int64 = 0

		erro := memLimit.Add(key, 4, 1*time.Second)
		if erro != nil {
			t.Error(erro)
		}

		// test get
		left, erro = memLimit.Get(key)

		if left != 3 || erro != nil {
			t.Errorf("%v %v should be 2 nil", left, erro)
		}

		// test get and limit
		sCnt := 0
		fCnt := 0
		var wg sync.WaitGroup
		for i := 0; i < 14; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				left, erro := memLimit.Get(key)
				if erro == nil && left > 0 {
					sCnt++
				} else {
					fCnt++
				}
			}()
		}

		wg.Wait()
		if sCnt != 2 {
			t.Errorf("%v,should be 2", sCnt)
		}

		// test set
		erro = memLimit.Set(key, 100, 1*time.Second)
		if erro != nil {
			t.Error(erro.Error())
		}

		left, erro = memLimit.Get(key)
		if erro != nil || left != 99 {
			t.Errorf("%v,%v should be 99,nil", left, erro)
		}

		// test timeout
		erro = memLimit.Set(key, 4, 4*time.Second)
		if erro != nil {
			t.Error(erro.Error())
		}

		left, erro = memLimit.Get(key)
		if erro != nil || left != 3 {
			t.Errorf("%v %v,should be 3,nil", left, erro)
		}

		time.Sleep(4 * time.Second)

		left, erro = memLimit.Get(key)
		if erro != nil || left != 3 {
			t.Errorf("%v %v,should be 3,nil", left, erro)
		}

		// test window
		erro = memLimit.Set(key, 4, 4*time.Second)
		if erro != nil {
			t.Error(erro.Error())
		}

		left, erro = memLimit.Get(key)
		if erro != nil || left != 3 {
			t.Errorf("%v %v,should be 3,nil", left, erro)
		}

		time.Sleep(1 * time.Second)

		left, erro = memLimit.Get(key)
		if erro != nil || left != 2 {
			t.Errorf("%v %v,should be 2,nil", left, erro)
		}

		time.Sleep(3 * time.Second)

		if wdwType == zlimiter.LimitMemFixWindow {
			left, erro = memLimit.Get(key)
			if erro != nil || left != 3 {
				t.Errorf("%v %v,should be 3,nil", left, erro)
			}
		} else if wdwType == zlimiter.LimitMemSlideWindow {
			left, erro = memLimit.Get(key)
			if erro != nil || left != 2 {
				t.Errorf("%v %v,should be 1,nil", left, erro)
			}
		}

		// test del
		memLimit.Del(key)
		left, erro = memLimit.Get(key)
		if left != zlimiter.ErrorReturnItemNotExist {
			t.Error("item should be not exist")
		}
	}
}

func TestToken(t *testing.T) {
	// create
	memLimit := zlimiter.NewLimiter(zlimiter.LimitMemToken)

	// test add
	key := "test"
	var left, max int64 = 0, 20
	erro := memLimit.Add(key, 4, 4*time.Second, max)
	if erro != nil {
		t.Errorf("error:%s", erro.Error())
	}

	time.Sleep(1 * time.Second)

	// test get
	left, erro = memLimit.Get(key)
	if left != 0 || erro != nil {
		t.Errorf("%v %v,should be 0,nil", left, erro)
	}

	//
	left, erro = memLimit.Get(key)
	if erro != nil || left != -1 {
		t.Errorf("%v %v,should be -1,nil", left, erro)
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
			left, erro := memLimit.Get(key)
			if erro == nil && left >= 0 {
				sCnt++
			} else {
				fCnt++
				if erro != nil {
					t.Logf("error:%s", erro.Error())
				}
			}
		}()
	}

	wg.Wait()

	//sCnt == 4
	if sCnt != 4 {
		t.Errorf("sCnt %d", sCnt)
	}

	// test set
	erro = memLimit.Set(key, 4, 2*time.Second, max)
	if erro != nil {
		t.Errorf("error %s", erro.Error())
	}

	time.Sleep(4 * time.Second)

	//
	left, erro = memLimit.Get(key)
	if left != 7 || erro != nil {
		t.Errorf("%v %v,should be 7,nil", left, erro)
	}

	// test overflow
	erro = memLimit.Set(key, 4, 4*time.Second, max)
	if erro != nil {
		t.Errorf("error %s", erro.Error())
	}

	time.Sleep(1 * time.Second)

	// reached,left == false 0
	left, erro = memLimit.Get(key)
	if left != 0 || erro != nil {
		t.Errorf("%v %v,should be 0,nil", left, erro)
	}

	time.Sleep(25 * time.Second)

	// reached,left == false 19
	left, erro = memLimit.Get(key)
	if left != 19 || erro != nil {
		t.Errorf("%v %v,should be 19,nil", left, erro)
	}

	// reached,left == false 18
	left, erro = memLimit.Get(key)
	if left != 18 || erro != nil {
		t.Errorf("%v %v,should be 18,nil", left, erro)
	}

	// test del
	memLimit.Del(key)
	left, erro = memLimit.Get(key)
	if left != zlimiter.ErrorReturnItemNotExist {
		t.Error("item should not be exist")
	}
}

func TestBucket(t *testing.T) {
	memLimit := zlimiter.NewLimiter(zlimiter.LimitMemBucket)

	// test add
	key := "test"
	var left, max, sCnt, fCnt int64 = 0, 20, 0, 0
	left = 0

	erro := memLimit.Add(key, 4, 4*time.Second, max)
	if erro != nil {
		t.Errorf("error:%s", erro.Error())
	}

	// test get and time span
	tm1 := time.Now()
	left, erro = memLimit.Get(key)
	if erro != nil || left != zlimiter.ErrorReturnBucket {
		t.Errorf("%v %v,should be -1,nil", left, erro)
	}

	//
	left, erro = memLimit.Get(key)
	if erro != nil || left != zlimiter.ErrorReturnBucket {
		t.Errorf("%v %v,should be -1,nil", left, erro)
	}
	tm2 := time.Now()

	// duration about 1s
	duSec := tm2.Sub(tm1).Seconds()
	if int64(duSec) != 1 {
		t.Errorf("errors:tm duration should be 1 sec")
	}

	// test get and limit
	var wg sync.WaitGroup
	for i := 0; i < 14; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			left, erro := memLimit.Get(key)
			if erro == nil && left == zlimiter.ErrorReturnBucket {
				sCnt++
			} else {
				fCnt++
			}
		}()
	}

	wg.Wait()
	if sCnt != 14 {
		t.Errorf("sCnt %d", sCnt)
	}

	// test set
	erro = memLimit.Set(key, 4, 8*time.Second, max)
	if erro != nil {
		t.Errorf("error %s", erro.Error())
	}

	//
	tm1 = time.Now()
	left, erro = memLimit.Get(key)
	if left != zlimiter.ErrorReturnBucket || erro != nil {
		t.Errorf("%v %v,should be false,-1,nil", left, erro)
	}

	//
	left, erro = memLimit.Get(key)
	if left != zlimiter.ErrorReturnBucket || erro != nil {
		t.Errorf("%v %v,should be false,-1,nil", left, erro)
	}
	tm2 = time.Now()

	duSec = tm2.Sub(tm1).Seconds()
	if int64(duSec) != 2 {
		t.Errorf("errors:tm duration should be 2 sec")
	}

	// test sleep and time span
	erro = memLimit.Set(key, 4, 8*time.Second, max)
	if erro != nil {
		t.Errorf("error %s", erro.Error())
	}

	//reached,left == false 0
	tm1 = time.Now()
	left, erro = memLimit.Get(key)
	if left != zlimiter.ErrorReturnBucket || erro != nil {
		t.Errorf("%v %v,should be false,-1,nil", left, erro)
	}

	time.Sleep(1500 * time.Millisecond)

	//
	left, erro = memLimit.Get(key)
	if left != zlimiter.ErrorReturnBucket || erro != nil {
		t.Errorf("%v %v,should be false,-1,nil", left, erro)
	}

	tm2 = time.Now()
	duMs := tm2.Sub(tm1).Nanoseconds() / 1e6
	if math.Abs(float64(int64(duMs)-2000)) >= 100 {
		t.Errorf("errors:tm duration should be 2s")
	}
	t.Logf("tm duration should be 2s,du:%v", duMs)

	// test del
	memLimit.Del(key)
	left, erro = memLimit.Get(key)
	if left != zlimiter.ErrorReturnItemNotExist {
		t.Errorf("%v %v,should not find the key", left, erro)
	}
}

func TestReal(t *testing.T) {

	var left, sucCnt, failCnt int64
	var erro error
	ipLocal := getLocalIP()

	var windowTypes = [4]int64{zlimiter.LimitMemFixWindow, zlimiter.LimitMemSlideWindow, zlimiter.LimitMemBucket, zlimiter.LimitMemToken}

	for _, wdwType := range windowTypes {
		memLimit := zlimiter.NewLimiter(int64(wdwType))
		sucCnt = 0
		failCnt = 0

		left, erro = memLimit.Get(ipLocal)
		if erro == nil && left == zlimiter.ErrorReturnItemNotExist {
			memLimit.Add(ipLocal, 10, 1*time.Second, 30)
		}

		if wdwType == zlimiter.LimitMemToken {
			time.Sleep(1 * time.Second)
		}

		var wg sync.WaitGroup
		for i := 0; i < 23; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				left, erro = memLimit.Get(ipLocal)
				if erro != nil {
					fmt.Println(erro)
					failCnt++
				} else if left >= 0 || (left == zlimiter.ErrorReturnBucket && wdwType == zlimiter.LimitMemBucket) {
					sucCnt++
				} else if left < 0 {
					failCnt++
				}
			}()
		}

		wg.Wait()
		if wdwType == zlimiter.LimitMemFixWindow {
			if sucCnt != 10 && failCnt != 13 {
				t.Errorf("sucCnt != 10 && failCnt != 13")
			}
		} else if wdwType == zlimiter.LimitMemSlideWindow {
			if sucCnt != 10 && failCnt != 13 {
				t.Errorf("sucCnt != 10 && failCnt != 13")
			}
		} else if wdwType == zlimiter.LimitMemBucket {
			if sucCnt != 23 {
				t.Errorf("sucCnt != 23")
			}
		} else if wdwType == zlimiter.LimitMemToken {
			if sucCnt != 10 && failCnt != 13 {
				t.Errorf("sucCnt != 10 && failCnt != 13")
			}
		}

	}
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		os.Exit(1)
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return ""
}
