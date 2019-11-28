package memory

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/labstack/gommon/log"
	"github.com/zzerroo/zlimiter/driver/common"
)

// item 限流基础信息
type item struct {
	Key      string        // 用于标识限流规则
	Limits   int64         // 时间段内访问限制数，与Limits同时构成Druation内只允许Limits访问限制语义
	Druation time.Duration // 时间段，与Limits同时构成Druation内只允许Limits访问限制语义
}

type fixWindowItem struct {
	item              //	固定窗口限流中基础信息
	Idx     int64     //	该窗口的索引
	CurCnt  int64     //	该窗口内已经消费掉的访问的数目
	StartAt time.Time //
}

type slideWindowItem struct {
	item                //	该滑动窗口访问条目
	ReqList []time.Time //	该滑动窗口内的请求列表
}

type tokenItem struct {
	item                   // 该token的基础信息
	Max          int64     //	允许产生的最大token数目，如果TkLeft+本请求段内产生token数目超过Max，多余的token将被丢弃
	StartAt      time.Time //
	TkLeft       int64     //	上次请求后剩余的token数目
	CreateRateNs float64   //	token产生的速率，以NS为单位
}

type bucketItem struct {
	item              // 限流基础信息
	StartAt time.Time //
	cteDuNS int64
}

// 桶限流计数信息
type bucketItemCnt struct {
	Max       int64         //	最大缓存请求数目
	WaitQueue chan struct{} //	请求缓存队列
}

// CacheFixWindow  固定窗口限流，以map存储所有限流规则，map的key为限流规则key
type CacheFixWindow struct {
	items map[string]fixWindowItem
	rwMut sync.RWMutex
}

// CacheSlideWindow 滑动敞口限流，以map存储所有限流规则，map的key为限流规则key
type CacheSlideWindow struct {
	items map[string]slideWindowItem
	rwMut sync.RWMutex
}

// Bucket 桶限流
type Bucket struct {
	items   map[string]bucketItem
	itemCnt sync.Map   //	sync map用于同步基于key的bucketItemCnt
	rwMut   sync.Mutex //	用于同步items的访问
}

// Token token限流
type Token struct {
	items map[string]tokenItem
	rwMut sync.RWMutex //	用于对items的同步访问
}

// Init 滑动限流窗口的初始化，创建相关map
func (c *CacheSlideWindow) Init(...interface{}) error {
	c.items = make(map[string]slideWindowItem)
	return nil
}

// Add 滑动窗口限流中新增一条限流规则。注意,如果相应key存在则会更新相关信息
//	Input :
//		key : 限流标识，用于唯一标识一条限流规则
//		limit : tmDuriation时间段内的限流数，与tmDuriation同时实现tmDuriation时间段内限流limit次的语义
//		tmDuriation : 时间段, 与limit同时实现tmDuriation时间段内限流limit次的语义
//		others : 未启用
//	Output :
//		error : 成功为nil，否则为具体错误信息
func (c *CacheSlideWindow) Add(key string, limits int64, tmDuriation time.Duration, others ...interface{}) error {
	c.rwMut.Lock()
	defer c.rwMut.Unlock()

	itemTmp := slideWindowItem{
		item: item{
			Key:      key,
			Limits:   limits,
			Druation: tmDuriation,
		},
		ReqList: make([]time.Time, 0), //请求队列初始长度为0
	}
	c.items[key] = itemTmp
	return nil
}

// Set 滑动窗口限流中重置或新增一条限流规则。注意,如果相应key存在则会更新相关信息
//	Input :
//		key : 限流标识，用于唯一标识一条限流规则
//		limits : tmDuriation时间段内的限流数，与tmDuriation同时实现tmDuriation时间段内限流limit次的语义
//		tmDuriation : 时间段, 与limit同时实现tmSpan时间段内限流limit次的语义
//		others : 未启用
//	Output :
//		error : 成功为nil，否则为具体错误信息
func (c *CacheSlideWindow) Set(key string, limits int64, tmDuriation time.Duration, others ...interface{}) error {
	c.rwMut.Lock()
	defer c.rwMut.Unlock()

	itemTmp := slideWindowItem{
		item: item{
			Key:      key,
			Limits:   limits,
			Druation: tmDuriation,
		},
		ReqList: make([]time.Time, 0), //请求队列初始长度为0
	}
	c.items[key] = itemTmp
	return nil
}

// Del 滑动窗口限流中删除key对应限流规则
//	Input :
//		key : 限流标识，用于标识要删除的限流规则
//	Output :
//		error : 成功为nil，否则为具体错误信息
func (c *CacheSlideWindow) Del(key string) error {
	c.rwMut.Lock()
	defer c.rwMut.Unlock()

	delete(c.items, key)
	return nil
}

// Get 滑动窗口限流中获取key对应剩余请求数。滑动窗口的限流维护了基于
//	时间的请求队列，每次请求会根据当前时间和规则对应的时间段(Duration)重新计算滑动窗口的起始时间，并根据
//	滑动窗口时间段内请求数目判断和返回相关限流信息
//	Input :
//		key : 限流标识，用于标识要获取限流相关信息的规则
//	Output :
//		int64 : 当前敞口中剩余请求数目，<-1000为错误，具体参见common.ErrorReturn*
//		error : 成功为nil，否则为具体错误信息
func (c *CacheSlideWindow) Get(key string) (int64, error) {
	defer func() {
		if p := recover(); p != nil {
			log.Errorf(common.ErrorUnknown)
		}
	}()

	c.rwMut.Lock()
	defer c.rwMut.Unlock()

	var itemTmp slideWindowItem
	var ok bool

	// 相关key不存在，返回错误common.ErrorInputParam
	if itemTmp, ok = c.items[key]; !ok {
		return common.ErrorReturnItemNotExist, nil
	}

	cur := time.Now()
	var left int64

	// 计算滑动窗口起点
	start := cur.Add(-1 * itemTmp.Druation)

	// 删除滑动窗口以外的数据
	for _, tm := range itemTmp.ReqList {
		if tm.Before(start) {
			itemTmp.ReqList = itemTmp.ReqList[1:]
		}
	}

	// 请求数已经超过了限制
	if int64(len(itemTmp.ReqList)) >= itemTmp.Limits {
		return common.ErrorReturnNoLeft, nil
	}

	// 追加当前请求到请求队列
	itemTmp.ReqList = append(itemTmp.ReqList, cur)
	left = itemTmp.Limits - int64(len(itemTmp.ReqList))
	c.items[key] = itemTmp
	return left, nil
}

// Init 创建CacheFixWindow中规则map
func (c *CacheFixWindow) Init(...interface{}) error {
	c.items = make(map[string]fixWindowItem)
	return nil
}

// Add 固定窗口限流中新增一条限流规则。同样,如果相应key存在则会更新相关信息。固定窗口限流新增规则时，
//	窗口索引值初始为0,StartAt为当前时间
//	Input :
//		key : 限流标识，用于唯一标识一条限流规则
//		limit : tmSpan时间段内的限流数，与tmSpan同时实现tmSpan时间段内限流limit次的语义
//		tmDuriation : 时间段, 与limit同时实现tmSpan时间段内限流limit次的语义
//		others : 未启用
//	Output :
//		error : 成功为nil，否则为具体错误信息
func (c *CacheFixWindow) Add(key string, limits int64, tmDuriation time.Duration, others ...interface{}) error {
	c.rwMut.Lock()
	defer c.rwMut.Unlock()

	itemTmp := fixWindowItem{
		Idx:    0,
		CurCnt: 0,
		item: item{
			Key:      key,
			Limits:   limits,
			Druation: tmDuriation,
		},
		StartAt: time.Now(),
	}
	c.items[key] = itemTmp
	return nil
}

// Set 固定窗口限流中重置或新增一条限流规则。注意,如果相应key存在则会更新相关信息，不管规则是否存在
//	Idx和CurCnt都会置为0
//	Input :
//		key : 限流标识，用于唯一标识一条限流规则
//		limits : tmDuriation时间段内的限流数，与tmDuriation同时实现tmDuriation时间段内限流limits次的语义
//		tmDuriation : 时间段, 与limits同时实现tmSpan时间段内限流limit次的语义
//		others : 未启用
//	Output :
//		error : 成功为nil，否则为具体错误信息
func (c *CacheFixWindow) Set(key string, limits int64, tmDuriation time.Duration, others ...interface{}) error {
	c.rwMut.Lock()
	defer c.rwMut.Unlock()

	itemTmp := fixWindowItem{
		Idx:    0,
		CurCnt: 0,
		item: item{
			Key:      key,
			Limits:   limits,
			Druation: tmDuriation,
		},
		StartAt: time.Now(),
	}
	c.items[key] = itemTmp
	return nil
}

// Del 删除固定窗口限流中一条限流规则
//	Input :
//		key : 限流标识，用于唯一标识一条限流规则
//	Output :
//		error : 成功为nil，否则为具体错误信息
func (c *CacheFixWindow) Del(key string) error {
	c.rwMut.Lock()
	defer c.rwMut.Unlock()

	delete(c.items, key)
	return nil
}

// Get 获得固定窗口限流中key对应规则的剩余请求数目。根据当前时间和规则创建或者
//	重置的时间间隔，Get会计算当前请求所在的窗口的索引，如果该索引和上次请求的索引相同，则判断为旧窗口 和旧请求数目
//	同时进行计数，否则则判断为新窗口 重新开始计数
//	Input :
//		key : 限流标识，用于标识要获取限流相关信息的规则
//	Output :
//		int64 : 当前窗口中剩余请求的数目, <-1000产生了错误，具体参见common.ErrorReturn*
//		error : 成功为nil，否则为具体错误信息
func (c *CacheFixWindow) Get(key string) (int64, error) {
	c.rwMut.Lock()
	defer c.rwMut.Unlock()

	var itemTmp fixWindowItem
	var ok bool

	// 如果相关key不存在，则返回ErrorInputParam
	if itemTmp, ok = c.items[key]; !ok {
		return common.ErrorReturnItemNotExist, nil
	}

	curTm := time.Now()
	startAt := itemTmp.StartAt

	// 根据已经逝去的时间段和限流的Druation,计算当前窗口的索引信息
	elapsed := curTm.Sub(startAt)
	idx := elapsed.Nanoseconds() / itemTmp.Druation.Nanoseconds()
	if idx != itemTmp.Idx { // 新窗口则重新开始计数
		itemTmp.Idx = idx
		itemTmp.CurCnt = 1

		c.items[key] = itemTmp
		return itemTmp.Limits - itemTmp.CurCnt, nil
	}

	// 已经存在的窗口，根据计CurCnt计数剩余请求数
	left := itemTmp.Limits - itemTmp.CurCnt - 1
	itemTmp.CurCnt = itemTmp.CurCnt + 1
	c.items[key] = itemTmp
	return left, nil
}

// Init 创建Token限流中对应的map
func (t *Token) Init(...interface{}) error {
	t.items = make(map[string]tokenItem)
	return nil
}

// Add Token限流中新增或者更新一条规则
//	Input :
//		key : 规则的唯一标识
//		limits : tmDuriation时间段内 访问限制数目
//		tmDuriation : 规则对应的时间段，与limits共同构成 tmDuriation时间段内限流limits次的语义
//		other : int64, 最大token数目，如果当前剩余的+新产生的token数目>other，则抛弃
//	Output :
//		error : 成功则为nil，否则为对应错误
func (t *Token) Add(key string, limits int64, tmDuriation time.Duration, others ...interface{}) error {
	t.rwMut.Lock()
	defer t.rwMut.Unlock()

	var max int64
	var ok bool

	// 校验请求参数，token中应只包含一个值:max
	if len(others) != 1 {
		return errors.New(common.ErrorInputParam)
	}

	max, ok = others[0].(int64)
	if !ok {
		return errors.New(common.ErrorInputParam)
	}

	// 计算token的产生速度
	createRateNs := (float64(limits) / float64(tmDuriation.Nanoseconds()))
	itemTmp := tokenItem{
		item: item{
			Key:      key,
			Limits:   limits,
			Druation: tmDuriation,
		},
		Max:          max,
		StartAt:      time.Now(),
		TkLeft:       0,
		CreateRateNs: createRateNs,
	}
	t.items[key] = itemTmp

	return nil
}

// Del 删除Token限流中一条规则
//	Input :
//		key : 规则的唯一标识
//	Output :
//		error : 成功则为nil，否则为对应错误
func (t *Token) Del(key string) error {
	t.rwMut.Lock()
	defer t.rwMut.Unlock()

	delete(t.items, key)
	return nil
}

// Set Token限流中更新或新增一条规则
//	Input :
//		key : 规则的唯一标识
//		limits : tmDuriation时间段内 访问限制数目
//		tmDuriation : 规则对应的时间段，与limits共同构成 tmDuriation时间段内限流limits次的语义
//		others : int64, 最大token数目，如果当前剩余的+新产生的token数目>other，则抛弃
//	Output :
//		error : 成功则为nil，否则为对应错误
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
		},
		StartAt:      time.Now(),
		Max:          max,
		TkLeft:       0,
		CreateRateNs: createRateNs,
	}
	t.items[key] = itemTmp
	return nil
}

// Get 获取Token限流中规则对应剩余请求数目。每次Get请求都会根据token
//	产生速度(CreateRateNs)，距离上次请求已经逝去时间（time.Now().Sub(StartAt)），上次请求剩余的token数目（TkLeft）,计算当前
//	剩余token数目，并返回限流相关信息
//	Input :
//		key : 规则的唯一标识
//	Output :
//		int64 : 规则剩余请求数目，<-1000为错误，具体参见common.ErrorReturn*
//		error : 成功则为nil，否则为对应错误
func (t *Token) Get(key string) (int64, error) {
	t.rwMut.Lock()
	defer t.rwMut.Unlock()

	// 相关规则不存在，则返回ErrorInputParam错误
	itemTmp, ok := t.items[key]
	if !ok {
		return common.ErrorReturnItemNotExist, nil
	}

	// 分别计算距离上次请求产生的token数目
	curTm := time.Now()

	// 距离上次请求后新产生的token数目
	tkCreated := float64((curTm.Sub(itemTmp.StartAt)).Nanoseconds()) * itemTmp.CreateRateNs
	// 计算当前的token数，min(新产生的token数+上次剩余的token数，max)
	tkCur := int64(math.Min(tkCreated+float64(itemTmp.TkLeft), float64(itemTmp.Max)))
	if tkCur > 0 {
		itemTmp.TkLeft = int64(tkCur) - 1
		itemTmp.StartAt = time.Now()
		t.items[key] = itemTmp

		return itemTmp.TkLeft, nil
	}

	return common.ErrorReturnNoLeft, nil
}

// Init Bucket限流创建相关map
func (b *Bucket) Init(...interface{}) error {
	b.items = make(map[string]bucketItem)
	return nil
}

// Add Bucket限流中新增一条规则，注意：如果规则已经存在，只会更新相关规则信息。注意：为实现请求数目
//	的限制，Bucket限流中有一个专门用于计数的sync.map
//	Input :
//		key : 规则的唯一标识
//		limits : 当前规则在规定时间内限制请求的数目
//		tmDuriation : 当前规则对应的时间段，与limits共同构成 tmDuriation时间段内限流limits次的语义
//		others : bucket限流中最大缓存请求数目
//	Output :
//		error : 成功则返回nil，否则为相关错误信息
func (b *Bucket) Add(key string, limits int64, tmDuriation time.Duration, others ...interface{}) error {
	var max int64
	var ok bool

	// bucket限流中others只包含max信息
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
		cteDuNS: tmDuriation.Nanoseconds() / limits,
	}

	b.rwMut.Lock()
	b.items[key] = itemTmp
	b.rwMut.Unlock()

	// 限流缓存数目信息
	itemCntTmp := bucketItemCnt{
		Max:       max,
		WaitQueue: make(chan struct{}, max),
	}

	b.itemCnt.Store(key, itemCntTmp)
	return nil
}

// Set Bucket限流中更新或者新增一条规则，注意：如果规则不存在，会新增一条规则
//	Input :
//		key : 规则的唯一标识
//		limits : 当前规则在规定时间内限制请求的数目
//		tmDuriation : 当前规则对应的时间段，与limits共同构成 tmDuriation时间段内限流limits次的语义
//		others : bucket限流中最大缓存请求数目
//	Output :
//		error : 成功则返回nil，否则为相关错误信息
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
		cteDuNS: tmDuriation.Nanoseconds() / limits,
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

// Del Bucket限流中删除一条规则
//	Input :
//		key : 规则的标识
//	Output :
//		error : 成功则返回nil，否则为相关错误信息
func (b *Bucket) Del(key string) error {
	b.rwMut.Lock()
	delete(b.items, key)
	b.rwMut.Unlock()

	b.itemCnt.Delete(key)
	return nil
}

// getSyncMap 获得一条规则的计数信息，为了便于处理，bucket中计数信息会单独存储
//	Input :
//		key : 规则的标识
//	Output :
//		bucketItemCnt : error为nil则返回规则的计数信息，否则为空结构
//		error : 成功则返回nil，否则为相关错误信息
func (b *Bucket) getSyncMap(key string) (bucketItemCnt, string) {
	itemCntTmpIn, ok := b.itemCnt.Load(key)
	if !ok {
		return bucketItemCnt{}, common.ErrorItemNotExist
	}

	itemCntTmp, ok := itemCntTmpIn.(bucketItemCnt)
	if !ok {
		return bucketItemCnt{}, common.ErrorUnknown
	}

	return itemCntTmp, ""
}

// writeTimeout 写一条数据至规则的等待队列中，规则的等待队列为带缓存的channel（长度为max）。
//	写队列时候如果能改直接写入，则认为队列不满，否则认定队列已满
//	Input :
//		ch : 规则队列，用于实现请求的计数访问
//	Output :
//		bool : true代表写入成功，否则代表写入失败
func (b *Bucket) writeTimeout(ch chan<- struct{}) bool {
	select {
	case ch <- struct{}{}:
		return true
	default:
		return false
	}
}

// Get 获取Bucket限流中规则剩余请求数目。宏观上所有的Get请求间会以固定的速率返回
//	同时如果请求数目超过了bucket的max限制，请求会被抛弃。Get的以channel实现缓存队列，以sleep实现请求间的固定间隔
//	Input :
//		key : 规则的唯一标识
//	Output :
//		int64 : 规则剩余请求数目，<-1000为错误，具体参见common.ErrorReturn*
//		error : 成功则为nil，否则为对应错误
func (b *Bucket) Get(key string) (int64, error) {
	// 获得bucketItemCnt及相应channel
	itemCntTmp, erro := b.getSyncMap(key)
	if erro == common.ErrorItemNotExist {
		return common.ErrorReturnItemNotExist, nil
	} else if erro == common.ErrorUnknown {
		return common.ErrorReturnNoMeans, errors.New(common.ErrorUnknown)
	}

	if false == b.writeTimeout(itemCntTmp.WaitQueue) {
		return common.ErrorReturnNoLeft, nil
	}

	b.rwMut.Lock()
	itemTmp, ok := b.items[key]
	if !ok {
		b.rwMut.Unlock()
		<-itemCntTmp.WaitQueue //请求出队
		return common.ErrorReturnItemNotExist, nil
	}

	var timeWait int64
	tmCur := time.Now()

	// 如果是规则不是创建后首次get请求，则计算需要sleep的时间
	if !itemTmp.StartAt.IsZero() {
		//cteDuNS := itemTmp.Druation.Nanoseconds() / itemTmp.Limits
		tmDuPassed := tmCur.Sub(itemTmp.StartAt)
		timeWait = itemTmp.cteDuNS - tmDuPassed.Nanoseconds()%itemTmp.cteDuNS

		if timeWait != 0 {
			time.Sleep(time.Duration(timeWait) * time.Nanosecond)
		}
	}

	itemTmp.StartAt = tmCur
	b.items[key] = itemTmp
	b.rwMut.Unlock()

	<-itemCntTmp.WaitQueue //请求出队
	return common.ErrorReturnBucket, nil
}
