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
		memLimit := zlimiter.NewLimiter(int64(wdwType))

		// test add
		key := "test"
		var reached bool = false
		var left int64 = 0

		erro := memLimit.Add(key, 4, 1*time.Second)
		if erro != nil {
			t.Error(erro)
		}

		// test get
		reached, left, erro = memLimit.Get(key)

		if reached == true || left != 3 || erro != nil {
			t.Errorf("%v %v %v should be false 2 nil", reached, left, erro)
		}

		// test get and limit
		sCnt := 0
		fCnt := 0
		var wg sync.WaitGroup
		for i := 0; i < 14; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				reached, _, erro := memLimit.Get(key)
				if erro == nil && reached == false {
					sCnt++
				} else {
					fCnt++
				}
			}()
		}

		wg.Wait()
		if sCnt != 3 {
			t.Errorf("%v,should be 3", sCnt)
		}

		// test set
		erro = memLimit.Set(key, 100, 1*time.Second)
		if erro != nil {
			t.Error(erro.Error())
		}

		reached, left, erro = memLimit.Get(key)
		if erro != nil || reached != false || left != 99 {
			t.Errorf("%v,%v,%v should be false,99,nil", reached, left, erro)
		}

		// test timeout
		erro = memLimit.Set(key, 4, 4*time.Second)
		if erro != nil {
			t.Error(erro.Error())
		}

		reached, left, erro = memLimit.Get(key)
		if erro != nil || reached != false || left != 3 {
			t.Errorf("%v %v %v,should be false,3,nil", reached, left, erro)
		}

		time.Sleep(4 * time.Second)

		reached, left, erro = memLimit.Get(key)
		if erro != nil || reached != false || left != 3 {
			t.Errorf("%v %v %v,should be false,3,nil", reached, left, erro)
		}

		// test window
		erro = memLimit.Set(key, 4, 4*time.Second)
		if erro != nil {
			t.Error(erro.Error())
		}

		reached, left, erro = memLimit.Get(key)
		if erro != nil || reached == true || left != 3 {
			t.Errorf("%v %v %v,should be false,3,nil", reached, left, erro)
		}

		time.Sleep(1 * time.Second)

		reached, left, erro = memLimit.Get(key)
		if erro != nil || reached == true || left != 2 {
			t.Errorf("%v %v %v,should be false,2,nil", reached, left, erro)
		}

		time.Sleep(3 * time.Second)

		if wdwType == zlimiter.LimitMemFixWindow {
			reached, left, erro = memLimit.Get(key)
			if erro != nil || reached == true || left != 3 {
				t.Errorf("%v %v %v,should be false,3,nil", reached, left, erro)
			}
		} else if wdwType == zlimiter.LimitMemSlideWindow {
			reached, left, erro = memLimit.Get(key)
			if erro != nil || reached == true || left != 2 {
				t.Errorf("%v %v %v,should be false,1,nil", reached, left, erro)
			}
		}

		// test del
		memLimit.Del(key)
		_, _, erro = memLimit.Get(key)
		if erro == nil {
			t.Error("item should be not exist")
		}
	}
}

func TestToken(t *testing.T) {
	// create
	memLimit := zlimiter.NewLimiter(zlimiter.LimitMemToken)

	// test add
	key := "test"
	var reached bool = false
	var left, max int64 = 0, 20
	erro := memLimit.Add(key, 4, 4*time.Second, max)
	if erro != nil {
		t.Errorf("error:%s", erro.Error())
	}

	time.Sleep(1 * time.Second)

	// test get
	reached, left, erro = memLimit.Get(key)
	if erro != nil {
		t.Errorf("error:%s", erro.Error())
	}
	if reached == true || left != 0 || erro != nil {
		t.Errorf("%v %v %v,should be false,0,nil", reached, left, erro)
	}

	//
	reached, left, erro = memLimit.Get(key)
	if erro != nil || reached == false || left != -1 {
		t.Errorf("%v %v %v,should be false,-1,nil", reached, left, erro)
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

	// test set
	erro = memLimit.Set(key, 4, 2*time.Second, max)
	if erro != nil {
		t.Errorf("error %s", erro.Error())
	}

	time.Sleep(4 * time.Second)

	//
	reached, left, erro = memLimit.Get(key)
	if reached != false || left != 7 || erro != nil {
		t.Errorf("%v %v %v,should be false,7,nil", reached, left, erro)
	}

	// test overflow
	erro = memLimit.Set(key, 4, 4*time.Second, max)
	if erro != nil {
		t.Errorf("error %s", erro.Error())
	}

	time.Sleep(1 * time.Second)

	// reached,left == false 0
	reached, left, erro = memLimit.Get(key)
	if reached != false || left != 0 || erro != nil {
		t.Errorf("%v %v %v,should be false,0,nil", reached, left, erro)
	}

	time.Sleep(25 * time.Second)

	// reached,left == false 19
	reached, left, erro = memLimit.Get(key)
	if reached != false || left != 19 || erro != nil {
		t.Errorf("%v %v %v,should be false,19,nil", reached, left, erro)
	}

	// reached,left == false 18
	reached, left, erro = memLimit.Get(key)
	if reached != false || left != 18 || erro != nil {
		t.Errorf("%v %v %v,should be false,18,nil", reached, left, erro)
	}

	// test del
	memLimit.Del(key)
	_, _, erro = memLimit.Get(key)
	if erro == nil {
		t.Error("item should not be exist")
	}
}

func TestBucket(t *testing.T) {
	memLimit := zlimiter.NewLimiter(zlimiter.LimitMemBucket)

	// test add
	key := "test"
	var reached bool = false
	var left, max, sCnt, fCnt int64 = 0, 20, 0, 0
	left = 0

	erro := memLimit.Add(key, 4, 4*time.Second, max)
	if erro != nil {
		t.Errorf("error:%s", erro.Error())
	}

	// test get and time span
	tm1 := time.Now()
	reached, left, erro = memLimit.Get(key)
	if erro != nil || left != -1 || reached != false {
		t.Errorf("%v %v %v,should be false,-1,nil", reached, left, erro)
	}

	//
	reached, left, erro = memLimit.Get(key)
	if erro != nil || reached != false || left != -1 {
		t.Errorf("%v %v %v,should be false,-1,nil", reached, left, erro)
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
	reached, left, erro = memLimit.Get(key)
	if reached != false || left != -1 || erro != nil {
		t.Errorf("%v %v %v,should be false,-1,nil", reached, left, erro)
	}

	//
	reached, left, erro = memLimit.Get(key)
	if reached != false || left != -1 || erro != nil {
		t.Errorf("%v %v %v,should be false,-1,nil", reached, left, erro)
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
	reached, left, erro = memLimit.Get(key)
	if reached != false || left != -1 || erro != nil {
		t.Errorf("%v %v %v,should be false,-1,nil", reached, left, erro)
	}

	time.Sleep(1500 * time.Millisecond)

	//
	reached, left, erro = memLimit.Get(key)
	if reached != false || left != -1 || erro != nil {
		t.Errorf("%v %v %v,should be false,-1,nil", reached, left, erro)
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
}
