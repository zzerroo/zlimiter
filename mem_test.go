package zlimiter_test

import (
	"sync"
	"testing"
	"time"
	"github.com/zzerroo/zlimiter"
)

func TestMem(t *testing.T) {

	// create
	memLimit, erro := zlimiter.NewLimiter(zlimiter.LIMIT_TYPE_MEM)
	if erro != nil {
		t.Errorf("error:%s", erro.Error())
	}

	// test insert
	key := "test"
	erro = memLimit.Add(key, 3, 4*time.Second)
	if erro != nil {
		t.Errorf("error:%s", erro.Error())
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
	erro = memLimit.Set(key, 100, 10*time.Second)
	if erro != nil {
		t.Errorf("error %s", erro.Error())
	}

	reach, left, erro := memLimit.Get(key)
	if erro != nil || reach != false || left != 99 {
		t.Errorf("reach %v,left %d, error %s", reach, left, erro.Error())
	}

	// test timeout
	erro = memLimit.Set(key, 4, 3*time.Second)
	if erro != nil {
		t.Errorf("error %s", erro.Error())
	}

	reach, left, erro = memLimit.Get(key)
	if erro != nil || reach != false || left != 3 {
		t.Errorf("reach %v,left %d, error %s", reach, left, erro.Error())
	}

	time.Sleep(4 * time.Second)
	reach, left, erro = memLimit.Get(key)
	if erro != nil || reach != false || left != 3 {
		t.Errorf("reach %v,left %d, error %s", reach, left, erro.Error())
	}

	// test del
	memLimit.Delete(key)
	_, _, erro = memLimit.Get(key)
	if erro == nil {
		t.Error("should not find the key")
	}

	t.Logf("reach %v,left %d", reach, left)
}
