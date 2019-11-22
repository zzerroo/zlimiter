package memory

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/zzerroo/zlimiter/driver/common"
)

type item struct {
	// Key
	Key string
	// Cnt in the window
	Limits int64
	// Size of window
	Druation time.Duration
	// Start time, in fix window this is the first add time, in slide window this is the last add or set time
	StartAt time.Time
	// Window index
	//Idx int64
}

type fixWindowItem struct {
	item
	Idx    int64
	CurCnt int64
}

type slideWindowItem struct {
	item
	ReqList []time.Time
}

type tokenItem struct {
	item
	Max          int64
	TkLeft       int64
	CreateRateNs float64
}

type bucketItem struct {
	item
}

type bucketItemCnt struct {
	Max       int64
	WaitQueue chan struct{}
}

// CacheFixWindow  ...
type CacheFixWindow struct {
	items map[string]fixWindowItem
	rwMut sync.RWMutex
}

// CacheSlideWindow ...
type CacheSlideWindow struct {
	items map[string]slideWindowItem
	rwMut sync.RWMutex
}

// Bucket ...
type Bucket struct {
	items   map[string]bucketItem
	itemCnt sync.Map
	rwMut   sync.Mutex
}

// Token ...
type Token struct {
	items map[string]tokenItem
	rwMut sync.RWMutex
}

// Init ...
func (c *CacheSlideWindow) Init(...interface{}) error {
	c.items = make(map[string]slideWindowItem)
	return nil
}

// Add ...
func (c *CacheSlideWindow) Add(key string, limits int64, tmDuriation time.Duration, _ ...interface{}) error {
	c.rwMut.Lock()
	defer c.rwMut.Unlock()

	itemTmp := slideWindowItem{
		item: item{
			Key:      key,
			Limits:   limits,
			Druation: tmDuriation,
		},
		ReqList: make([]time.Time, 0),
	}
	c.items[key] = itemTmp
	return nil
}

// Set ...
func (c *CacheSlideWindow) Set(key string, limits int64, tmDuriation time.Duration, _ ...interface{}) error {
	c.rwMut.Lock()
	defer c.rwMut.Unlock()

	itemTmp := slideWindowItem{
		item: item{
			Key:      key,
			Limits:   limits,
			Druation: tmDuriation,
		},
		ReqList: make([]time.Time, 0),
	}
	c.items[key] = itemTmp
	return nil
}

// Del ...
func (c *CacheSlideWindow) Del(key string) error {
	c.rwMut.Lock()
	defer c.rwMut.Unlock()

	delete(c.items, key)
	return nil
}

// Get ...
func (c *CacheSlideWindow) Get(key string) (bool, int64, error) {
	c.rwMut.Lock()
	defer c.rwMut.Unlock()

	var itemTmp slideWindowItem
	var ok bool

	if itemTmp, ok = c.items[key]; !ok {
		return false, -1, errors.New(common.ErrorInputParam)
	}

	cur := time.Now()
	var left int64
	// 滑动窗口起点
	start := cur.Add(-1 * itemTmp.Druation)

	// 删除窗口以外的数据
	for _, tm := range itemTmp.ReqList {
		if tm.Before(start) {
			itemTmp.ReqList = itemTmp.ReqList[1:]
		}
	}

	if int64(len(itemTmp.ReqList)) >= itemTmp.Limits {
		return true, -1, nil
	}

	itemTmp.ReqList = append(itemTmp.ReqList, cur)
	left = itemTmp.Limits - int64(len(itemTmp.ReqList))
	c.items[key] = itemTmp
	return left < 0, left, nil
}

// Init ...
func (c *CacheFixWindow) Init(...interface{}) error {
	c.items = make(map[string]fixWindowItem)
	return nil
}

// Add ...
func (c *CacheFixWindow) Add(key string, limits int64, tmDuriation time.Duration, _ ...interface{}) error {
	c.rwMut.Lock()
	defer c.rwMut.Unlock()

	itemTmp := fixWindowItem{
		Idx:    0,
		CurCnt: 0,
		item: item{
			Key:      key,
			Limits:   limits,
			Druation: tmDuriation,
			StartAt:  time.Now()},
	}
	c.items[key] = itemTmp
	return nil
}

// Set this will reset the idx
func (c *CacheFixWindow) Set(key string, limits int64, tmDuriation time.Duration, _ ...interface{}) error {
	c.rwMut.Lock()
	defer c.rwMut.Unlock()

	itemTmp := fixWindowItem{
		Idx:    0,
		CurCnt: 0,
		item: item{
			Key:      key,
			Limits:   limits,
			Druation: tmDuriation,
			StartAt:  time.Now()},
	}
	c.items[key] = itemTmp
	return nil
}

// Del ...
func (c *CacheFixWindow) Del(key string) error {
	c.rwMut.Lock()
	defer c.rwMut.Unlock()

	delete(c.items, key)
	return nil
}

// Get ...
func (c *CacheFixWindow) Get(key string) (bool, int64, error) {
	c.rwMut.Lock()
	defer c.rwMut.Unlock()

	var itemTmp fixWindowItem
	var ok bool

	if itemTmp, ok = c.items[key]; !ok {
		return false, -1, errors.New(common.ErrorInputParam)
	}

	curTm := time.Now()
	startAt := itemTmp.StartAt

	elapsed := curTm.Sub(startAt)
	idx := elapsed.Nanoseconds() / itemTmp.Druation.Nanoseconds()
	if idx != itemTmp.Idx {
		itemTmp.Idx = idx
		itemTmp.CurCnt = 1

		c.items[key] = itemTmp
		return false, itemTmp.Limits - itemTmp.CurCnt, nil
	}

	left := itemTmp.Limits - itemTmp.CurCnt - 1
	itemTmp.CurCnt = itemTmp.CurCnt + 1
	c.items[key] = itemTmp
	return left < 0, left, nil
}

// Init ...
func (t *Token) Init(...interface{}) error {
	t.items = make(map[string]tokenItem)
	return nil
}

// Add ...
func (t *Token) Add(key string, limits int64, tmDuriation time.Duration, others ...interface{}) error {
	t.rwMut.Lock()
	defer t.rwMut.Unlock()

	var max int64
	var ok bool

	// through three call
	if len(others) != 1 {
		return errors.New(common.ErrorInputParam)
	}

	max, ok = others[0].(int64)
	if !ok {
		return errors.New(common.ErrorInputParam)
	}

	createRateNs := (float64(limits) / float64(tmDuriation.Nanoseconds()))
	itemTmp := tokenItem{
		item: item{
			Key:      key,
			Limits:   limits,
			Druation: tmDuriation,
			StartAt:  time.Now(),
		},
		Max:          max,
		TkLeft:       0,
		CreateRateNs: createRateNs,
	}
	t.items[key] = itemTmp

	return nil
}

// Del ...
func (t *Token) Del(key string) error {
	t.rwMut.Lock()
	defer t.rwMut.Unlock()

	delete(t.items, key)
	return nil
}

// Set ...
func (t *Token) Set(key string, limits int64, tmDuriation time.Duration, others ...interface{}) error {
	t.rwMut.Lock()
	defer t.rwMut.Unlock()

	var max int64
	var ok bool

	if len(others) == 1 {
		max, ok = others[0].(int64)
		if !ok {
			return errors.New(common.ErrorInputParam)
		}
	}

	createRateNs := (float64(limits) / float64(tmDuriation.Nanoseconds()))
	itemTmp := tokenItem{
		item: item{
			Key:      key,
			Limits:   limits,
			Druation: tmDuriation,
			StartAt:  time.Now(),
		},
		Max:          max,
		TkLeft:       0,
		CreateRateNs: createRateNs,
	}
	t.items[key] = itemTmp
	return nil
}

// Get ...
func (t *Token) Get(key string) (bool, int64, error) {
	t.rwMut.Lock()
	defer t.rwMut.Unlock()

	// no exist
	itemTmp, ok := t.items[key]
	if !ok {
		return false, -1, errors.New(common.ErrorInputParam)
	}

	// Cal coumt of token created between StartAt and now
	curTm := time.Now()
	tkCreated := float64((curTm.Sub(itemTmp.StartAt)).Nanoseconds()) * itemTmp.CreateRateNs
	tkCur := int64(math.Min(tkCreated+float64(itemTmp.TkLeft), float64(itemTmp.Max)))
	if tkCur > 0 {
		itemTmp.TkLeft = int64(tkCur) - 1
		itemTmp.StartAt = time.Now()
		t.items[key] = itemTmp

		return false, itemTmp.TkLeft, nil
	}
	return true, -1, nil
}

// Init ...
func (b *Bucket) Init(...interface{}) error {
	b.items = make(map[string]bucketItem)
	return nil
}

// Add ...
func (b *Bucket) Add(key string, limits int64, tmDuriation time.Duration, others ...interface{}) error {
	var max int64
	var ok bool

	if len(others) == 1 {
		max, ok = others[0].(int64)
		if !ok {
			return errors.New(common.ErrorInputParam)
		}
	}

	itemTmp := bucketItem{
		item: item{
			Key:      key,
			Limits:   limits,
			Druation: tmDuriation,
		},
	}

	b.rwMut.Lock()
	b.items[key] = itemTmp
	b.rwMut.Unlock()

	itemCntTmp := bucketItemCnt{
		Max:       max,
		WaitQueue: make(chan struct{}, max),
	}

	b.itemCnt.Store(key, itemCntTmp)
	return nil
}

// Set ...
func (b *Bucket) Set(key string, limits int64, tmDuriation time.Duration, others ...interface{}) error {
	var max int64
	var ok bool

	if len(others) == 1 {
		max, ok = others[0].(int64)
		if !ok {
			return errors.New(common.ErrorInputParam)
		}
	}

	itemTmp := bucketItem{
		item: item{
			Key:      key,
			Limits:   limits,
			Druation: tmDuriation,
		},
	}
	b.rwMut.Lock()
	b.items[key] = itemTmp
	b.rwMut.Unlock()

	itemCntTmp := bucketItemCnt{
		Max:       max,
		WaitQueue: make(chan struct{}, max),
	}
	b.itemCnt.Store(key, itemCntTmp)
	return nil
}

// Del ...
func (b *Bucket) Del(key string) error {
	b.rwMut.Lock()
	delete(b.items, key)
	b.rwMut.Unlock()

	b.itemCnt.Delete(key)
	return nil
}

func (b *Bucket) getSyncMap(key string) (bucketItemCnt, error) {
	itemCntTmpIn, ok := b.itemCnt.Load(key)
	if !ok {
		return bucketItemCnt{}, errors.New(common.ErrorItemNotExist)
	}

	itemCntTmp, ok := itemCntTmpIn.(bucketItemCnt)
	if !ok {
		return bucketItemCnt{}, errors.New(common.ErrorUnknown)
	}

	return itemCntTmp, nil
}

func (b *Bucket) writeTimeout(ch chan<- struct{}) bool {
	select {
	case ch <- struct{}{}:
		return true
	default:
		return false
	}
}

// Get ...
func (b *Bucket) Get(key string) (bool, int64, error) {
	itemCntTmp, erro := b.getSyncMap(key)
	if erro != nil {
		return false, -1, errors.New(erro.Error())
	}

	if false == b.writeTimeout(itemCntTmp.WaitQueue) {
		return false, -1, errors.New(common.ErrorReqOverFlow)
	}

	b.rwMut.Lock()
	itemTmp, ok := b.items[key]
	if !ok {
		b.rwMut.Unlock()
		<-itemCntTmp.WaitQueue
		return false, -1, errors.New(common.ErrorUnknown)
	}

	var timeWait int64
	tmCur := time.Now()

	if !itemTmp.StartAt.IsZero() {
		cteDuNS := itemTmp.Druation.Nanoseconds() / itemTmp.Limits
		tmDuPassed := tmCur.Sub(itemTmp.StartAt)
		timeWait = cteDuNS - tmDuPassed.Nanoseconds()%cteDuNS

		if timeWait != 0 {
			time.Sleep(time.Duration(timeWait) * time.Nanosecond)
		}
	}

	itemTmp.StartAt = tmCur
	b.items[key] = itemTmp
	b.rwMut.Unlock()

	<-itemCntTmp.WaitQueue
	return false, -1, nil
}
