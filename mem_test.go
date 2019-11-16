package zlimiter_test

import (
	"math"
	"sync"
	"testing"
	"time"

	"github.com/zzerroo/zlimiter"
)

func TestMem(t *testing.T) {
	var windowTypes = [2]int64{zlimiter.LimitMemFixWindow, zlimiter.LimitMemSlideWindow}
	for _, wdwType := range windowTypes {
		memLimit, erro := zlimiter.NewLimiter(int64(wdwType))
		if erro != nil {
			t.Errorf("error:%s", erro.Error())
		}

		// test add
		key := "test"
		var reached bool = false
		var left int64 = 0

		erro = memLimit.Add(key, 4, 1*time.Second)
		if erro != nil {
			t.Errorf("error:%s", erro.Error())
		}

		// test get
		reached, left, erro = memLimit.Get(key)
		if erro != nil {
			t.Errorf("error:%s", erro.Error())
		}

		if reached == true || left != 3 {
			t.Errorf("error:result should be false and left should be 2")
		}

		// test get and limit
		sCnt := 0
		fCnt := 0
		var wg sync.WaitGroup
		for i := 0; i < 14; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				reach, _, erro := memLimit.Get(key)
				if erro == nil && reach == false {
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
		if sCnt != 3 {
			t.Errorf("sCnt %d", sCnt)
		}

		// test set
		erro = memLimit.Set(key, 100, 1*time.Second)
		if erro != nil {
			t.Errorf("error %s", erro.Error())
		}

		reach, left, erro := memLimit.Get(key)
		if erro != nil || reach != false || left != 99 {
			t.Errorf("reach %v,left %d, error %v", reach, left, erro)
		}

		// test timeout
		erro = memLimit.Set(key, 4, 4*time.Second)
		if erro != nil {
			t.Errorf("error %s", erro.Error())
		}

		reach, left, erro = memLimit.Get(key)
		if erro != nil || reach != false || left != 3 {
			t.Errorf("reach %v,left %d, error %v", reach, left, erro)
		}

		time.Sleep(4 * time.Second)

		reach, left, erro = memLimit.Get(key)
		if erro != nil || reach != false || left != 3 {
			t.Errorf("reach %v,left %d, error %v", reach, left, erro)
		}

		// test window
		erro = memLimit.Set(key, 4, 5*time.Second)
		if erro != nil {
			t.Errorf("error %s", erro.Error())
		}

		time.Sleep(13 * time.Second)

		reach, left, erro = memLimit.Get(key)
		if erro != nil || reach == true || left != 3 {
			t.Errorf("reach %v,left %d, error %v", reach, left, erro)
		}

		reach, left, erro = memLimit.Get(key)
		if erro != nil || reach == true || left != 2 {
			t.Errorf("reach %v,left %d, error %v", reach, left, erro)
		}

		time.Sleep(3 * time.Second)

		if wdwType == zlimiter.LimitMemFixWindow {
			reach, left, erro = memLimit.Get(key)
			if erro != nil || reach == true || left != 3 {
				t.Errorf("reach %v,left %d, error %v", reach, left, erro)
			}
		} else if wdwType == zlimiter.LimitMemSlideWindow {
			reach, left, erro = memLimit.Get(key)
			if erro != nil || reach == true || left != 1 {
				t.Errorf("reach %v,left %d, error %v", reach, left, erro)
			}
		}

		// test del
		memLimit.Del(key)
		_, _, erro = memLimit.Get(key)
		if erro == nil {
			t.Error("should not find the key")
		}

		t.Logf("reach %v,left %d", reach, left)
	}
}

func TestToken(t *testing.T) {
	// create
	memLimit, erro := zlimiter.NewLimiter(zlimiter.LimitMemToken)
	if erro != nil {
		t.Errorf("error:%s", erro.Error())
	}

	// test add
	key := "test"
	var reached bool = false
	var left, max int64 = 0, 20
	erro = memLimit.Add(key, 4, 4*time.Second, max)
	if erro != nil {
		t.Errorf("error:%s", erro.Error())
	}

	time.Sleep(1 * time.Second)

	// reached,left == false,0
	reached, left, erro = memLimit.Get(key)
	if erro != nil {
		t.Errorf("error:%s", erro.Error())
	}
	if reached == true || left != 0 {
		t.Errorf("errors:reach should be false and left should be 0")
	}

	// reached,left == true,-1
	reached, left, erro = memLimit.Get(key)
	if erro != nil || reached == false {
		t.Errorf("errors:reach should be false and left should be 0")
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
			reach, _, erro := memLimit.Get(key)
			if erro == nil && reach == false {
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
	t.Logf("should 4, sCnt:%v", sCnt)

	// test set
	erro = memLimit.Set(key, 4, 2*time.Second, max)
	if erro != nil {
		t.Errorf("error %s", erro.Error())
	}

	time.Sleep(4 * time.Second)

	// reached,left == false,7
	reached, left, erro = memLimit.Get(key)
	if reached != false || left != 7 {
		t.Errorf("reach %v,left %d", reached, left)
		if erro != nil {
			t.Errorf("%v", erro.Error())
		}
	}

	// test overflow
	erro = memLimit.Set(key, 4, 4*time.Second, max)
	if erro != nil {
		t.Errorf("error %s", erro.Error())
	}

	time.Sleep(1 * time.Second)

	// reached,left == false 0
	reached, left, erro = memLimit.Get(key)
	if reached != false || left != 0 {
		t.Errorf("reach %v,left %d", reached, left)
		if erro != nil {
			t.Errorf("%v", erro.Error())
		}
	}

	time.Sleep(25 * time.Second)

	// reached,left == false 19
	reached, left, erro = memLimit.Get(key)
	if reached != false || left != 19 {
		t.Errorf("reach %v,left %d", reached, left)
		if erro != nil {
			t.Errorf("%v", erro.Error())
		}
	}
	t.Logf("should false,19 reached:%v,left:%v", reached, left)

	// reached,left == false 18
	reached, left, erro = memLimit.Get(key)
	if reached != false || left != 18 {
		t.Errorf("reach %v,left %d", reached, left)
		if erro != nil {
			t.Errorf("%v", erro.Error())
		}
	}

	// test del
	memLimit.Del(key)
	_, _, erro = memLimit.Get(key)
	if erro == nil {
		t.Error("should not find the key")
	}
}

func TestBucket(t *testing.T) {
	memLimit, erro := zlimiter.NewLimiter(zlimiter.LimitMemBucket)
	if erro != nil {
		t.Errorf("error:%s", erro.Error())
	}

	// test add
	key := "test"
	var reached bool = false
	var left, max, sCnt, fCnt int64 = 0, 20, 0, 0
	left = 0

	erro = memLimit.Add(key, 4, 4*time.Second, max)
	if erro != nil {
		t.Errorf("error:%s", erro.Error())
	}

	// reached，left == false,-1
	tm1 := time.Now()
	reached, left, erro = memLimit.Get(key)
	if erro != nil {
		t.Errorf("error:%s", erro.Error())
	}
	t.Logf("should false,-1 reached:%v,left:%v", reached, left)

	if reached == true || left != -1 {
		t.Errorf("errors:reach should be false and left should be 0")
	}

	// reached,left == true,0
	reached, left, erro = memLimit.Get(key)
	if erro != nil || reached != false || left != -1 {
		t.Errorf("errors:reach should be false and left should be 0")
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
			reach, _, erro := memLimit.Get(key)
			if erro == nil && reach == false && left == -1 {
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
	if sCnt != 14 {
		t.Errorf("sCnt %d", sCnt)
	}
	t.Logf("should 14, sCnt:%v", sCnt)

	// test set
	erro = memLimit.Set(key, 4, 8*time.Second, max)
	if erro != nil {
		t.Errorf("error %s", erro.Error())
	}

	// reached,left == false,-1
	tm1 = time.Now()
	reached, left, erro = memLimit.Get(key)
	if reached != false || left != -1 {
		t.Errorf("reach %v,left %d", reached, left)
		if erro != nil {
			t.Errorf("%v", erro.Error())
		}
	}

	// reached,left == false,-1
	reached, left, erro = memLimit.Get(key)
	if reached != false || left != -1 {
		t.Errorf("reach %v,left %d", reached, left)
		if erro != nil {
			t.Errorf("%v", erro.Error())
		}
	}
	tm2 = time.Now()

	duSec = tm2.Sub(tm1).Seconds()
	if int64(duSec) != 2 {
		t.Errorf("errors:tm duration should be 2 sec")
	}

	// test overflow
	max = 5
	erro = memLimit.Set(key, 4, 2*time.Second, max)
	if erro != nil {
		t.Errorf("error %s", erro.Error())
	}

	sCnt = 0
	fCnt = 0
	dataLen := 15
	for i := 0; i < dataLen; i++ {
		wg.Add(1)
		i := i
		go func(idx int) {
			defer wg.Done()
			reached, left, erro = memLimit.Get(key)
			if erro != nil {
				fCnt++
			} else {
				sCnt++
			}
		}(i)
	}

	wg.Wait()
	if fCnt != int64(dataLen)-max-1 { // ????
		t.Error("error: failed cnt should be 5")
	}

	erro = memLimit.Set(key, 4, 8*time.Second, max)
	if erro != nil {
		t.Errorf("error %s", erro.Error())
	}

	//reached,left == false 0
	tm1 = time.Now()
	reached, left, erro = memLimit.Get(key)
	if reached != false || left != -1 {
		t.Errorf("reach %v,left %d", reached, left)
		if erro != nil {
			t.Errorf("%v", erro.Error())
		}
	}

	time.Sleep(1500 * time.Millisecond)

	// reached,left == false,0
	reached, left, erro = memLimit.Get(key)
	if reached != false || left != -1 {
		t.Errorf("reach %v,left %d", reached, left)
		if erro != nil {
			t.Errorf("%v", erro.Error())
		}
	}

	tm2 = time.Now()
	duMs := tm2.Sub(tm1).Nanoseconds() / 1e6
	if math.Abs(float64(int64(duMs)-2000)) >= 100 {
		t.Errorf("errors:tm duration should be 2s")
	}
	t.Logf("tm duration should be 2s,du:%v", duMs)

	// test del
	memLimit.Del(key)
	_, _, erro = memLimit.Get(key)
	if erro == nil {
		t.Error("should not find the key")
	}

	t.Logf("reach %v,left %d", reached, left)
}
