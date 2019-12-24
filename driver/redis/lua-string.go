package redis

const (
	// FixAddStr : 固定窗口限流Add脚本, 新增一条固定窗口限流规则。规则中idx为当前窗口的索引值，limit为限额大小，current为当前时间(微妙)，
	//	wdwcnt为当前窗口已放行的访问的量
	// Input :
	// 	key : 规则对应的key，2d1b74349305508b-fix为其中前缀
	// 	limit : duration时间内的限额数目
	// 	duration : 时间段
	// 	current: 当前时间 (μs)
	FixAddStr = `
local key = KEYS[1] --key
local limit = tonumber(ARGV[1]) --限流大小
local duration = tonumber(ARGV[2]) --时长
local current = tonumber(ARGV[3]) --current timestamp
redis.call('HMSET','2d1b74349305508b-fix'..key,'limit',limit,'duration',duration,'idx',0,'start',current,'wdwcnt',0)
`
	// FixGetStr : 固定窗口限流get脚本, 固定窗口限流将整个时间段划分为长度为duration的固定窗口，并为每个窗口维护了idx、wdwcnt分别表示
	//	当前窗口对应的索引和当前窗口期中已经放行的请求数，并通过idx、wdwcnt实现窗口的定位和限额
	// Input:
	//  key: 规则ID
	//	current: 当前时间 (μs)
	// Output:
	// 0 : ok，规则可以放行
	// -1 : 规则在当前窗口已经超过了限额数
	// -2 : 相关的规则不存在
	FixGetStr = `
local key = KEYS[1] --key
local current = tonumber(ARGV[1]) --current timestamp
local limitInfos = redis.call('HMGET','2d1b74349305508b-fix'..key,'limit','duration','start','idx','wdwcnt')
	
local limit = tonumber(limitInfos[1])
local duration = tonumber(limitInfos[2])
local start = tonumber(limitInfos[3])
local idx = tonumber(limitInfos[4])
local curWdwCnt = tonumber(limitInfos[5])

if (limit == nil or duration == nil or start == nil or idx == nil or curWdwCnt == nil) then
    return -2
end
	
local retValue = 0
local curlIdx = math.ceil((current - start)/duration)
if (curlIdx ~= idx) then
	retValue = limit - 1
	idx = curlIdx
	curWdwCnt = 1
else
	curWdwCnt = curWdwCnt + 1
	if (curWdwCnt > limit) then
		retValue = -1
	else
		retValue = limit - curWdwCnt
	end
end

redis.call('HMSET','2d1b74349305508b-fix'..key,'idx',curlIdx,'wdwcnt',curWdwCnt)
return retValue
`

	// FixSetStr : 固定窗口限流set脚本，具体请求参见FixAddStr
	FixSetStr = FixAddStr

	// FixDelStr : 固定窗口限流删除脚本，删除一条规则对应的信息
	// Input:
	// 		key : 规则对应的key
	FixDelStr = `
local key = KEYS[1] --key
redis.call('HDEL','2d1b74349305508b-fix'..key,'limit','duration','idx','start','wdwcnt')
`
	// SlideAddStr : 滑动窗口限流add脚本，新增一条滑动窗口限流规则。规则中limit、duration表示在duration时间段内限制访问limit次，
	//	为了提高效率，滑动窗口采用了sortedset进行窗口的滑动(score为时间点)，为了保证member的唯一性 还维护了递增的属性idx
	// Input:
	//	key : 规则id，实际存储的key前缀为2d1b74349305508b-slide-h，2d1b74349305508b-slide-z
	// 	limit : 限额数
	// 	duration : 时间段
	// 	current : 当前时间 (μs)
	SlideAddStr = `
local key = KEYS[1] --key
local limit = tonumber(ARGV[1]) --限流大小
local duration = tonumber(ARGV[2]) --时长
local current = tonumber(ARGV[3]) --current timestamp
redis.call('HMSET','2d1b74349305508b-slide-h'..key,'limit',limit,'duration',duration,'idx',0)
redis.call('ZREMRANGEBYRANK','2d1b74349305508b-slide-z'..key,0,-1)
`
	// SlideSetStr : 滑动窗口set脚本，参见SlideAddStr
	SlideSetStr = `
local key = KEYS[1] --key
local limit = tonumber(ARGV[1]) --限流大小
local duration = tonumber(ARGV[2]) --时长
local current = tonumber(ARGV[3]) --current timestamp
redis.call('HMSET','2d1b74349305508b-slide-h'..key,'limit',limit,'duration',duration)
redis.call('ZREMRANGEBYRANK','2d1b74349305508b-slide-z'..key,0,-1)
`

	// SlideDelStr : 滑动窗口del脚本，删除一条限流规则
	// Input:
	//	key : 规则id
	SlideDelStr = `
local key = KEYS[1] --key
redis.call('HDEL','2d1b74349305508b-slide-h'..key,'limit','duration','idx')
redis.call('ZREMRANGEBYRANK','2d1b74349305508b-slide-z'..key,0,-1)
`
	// SlideGetStr : 滑动窗口get脚本, 脚本利用了sortedsort来实现基于时间的排序，每次get请求会根据当前时间current和规则对应的时间段duration 计算当前
	//	窗口的开始时间start，并根据sortedset中score值计算当前窗口已放行的请求数cnt，如果cnt>limit(限额数) 则返回-1，否则返回0表示放行
	// Input:
	// 	key : 规则key
	// 	current : 当前时间 (μs)
	// Output:
	//	0 : ok，放行
	//	-1 : 请求数目已经超过限额
	//	-2 : 对应的规则key不存在
	SlideGetStr = `
local key = KEYS[1] --key
local current = tonumber(ARGV[1]) --current timestamp
local limitInfos = redis.call('HMGET','2d1b74349305508b-slide-h'..key,'limit','duration','idx')
		
local limit = tonumber(limitInfos[1])
local duration = tonumber(limitInfos[2])
local curIdx = tonumber(limitInfos[3])

if (limit == nil or duration == nil or curIdx == nil) then
	return -2
end

local start = current - duration
local cnt = redis.call('ZCOUNT','2d1b74349305508b-slide-z'..key, start, 2147483647000000)

local retValue = 0
if (cnt >= limit) then
	retValue =  -1
else
	redis.call('ZADD','2d1b74349305508b-slide-z'..key, current, curIdx + 1)
	retValue = limit - cnt - 1
end

redis.call('HSET','2d1b74349305508b-slide-h'..key,'idx',curIdx + 1)

-- delete old data
local idxs = redis.call('ZRANGEBYSCORE','2d1b74349305508b-slide-z'..key, 0, '('..start )
for i=1,#idxs do
 	redis.call('ZREM','2d1b74349305508b-slide-z'..key, idxs[i])
end
return retValue
`
	// BucketAddStr : 桶限流add脚本，新增一条桶限流规则。limit、duration表示duration时间段内限制访问limit次语义，max表示最大缓存请求数，waitcnt表示当前等待的
	//	请求数目，span表示相邻两个请求需要等待的时间（用于实现宏观的匀速访问）
	// Input :
	//	key : 规则ID，实际存储的key都具有2d1b74349305508b-bucket前缀
	//	limit : 规定时间段的限额数
	//	duration : 时间段，与limit同时实现duration内限制访问limit次
	//	current : 当前时间 (μs)
	//	max : 桶的容量，能够缓存的最大请求数目
	BucketAddStr = `
local key = KEYS[1] --key
local limit = tonumber(ARGV[1]) --限流大小
local duration = tonumber(ARGV[2]) --时长
local current = tonumber(ARGV[3]) --current timestamp
local max = tonumber(ARGV[4]) -- max
local span = duration/limit
redis.call('HMSET','2d1b74349305508b-bucket'..key,'limit',limit,'duration',duration,'span',span,'last',0,'max',max,'waitcnt',0)
`
	// BucketGetStr : bucket限流get脚本。get脚本用于从桶限流中获得规则对应信息，包括当前请求是否可放行、是否存在等
	//	桶限流将当前等待的请求数目存至waitcnt属性中，每次get请求前都需要检查该变量是否已经超过了max值，同时get
	//	请求中也对waitcnt进行了二次校验
	// Input :
	//	key : 规则id
	//	current : 当前时间 (μs)
	// Output:
	//	0 : ok
	// -1 : overflow
	// -2 : key not exist
	BucketGetStr = `
local key = KEYS[1] --key
local current = tonumber(ARGV[1]) --current timestamp
local limitInfos = redis.call('HMGET','2d1b74349305508b-bucket'..key,'span','max','last','waitcnt')
	
local span = tonumber(limitInfos[1])
local max = tonumber(limitInfos[2])
local last = tonumber(limitInfos[3])
local waitCnt = tonumber(limitInfos[4])

if (span == nil or max == nil or last == nil or waitCnt == nil) then
    return -2
end

local retValue = 0
if (waitCnt > max) then
	waitCnt = waitCnt - 1
	redis.call('HMSET','2d1b74349305508b-bucket'..key,'last',current,'waitcnt',waitCnt)
	return -1
end

waitCnt = waitCnt - 1
if (last == 0) then
	redis.call('HMSET','2d1b74349305508b-bucket'..key,'last',current,'waitcnt',waitCnt)
	return 0
end

local tmWait = 0
if (last <= current) then
	if (last + span > current) then
		redis.call('HMSET','2d1b74349305508b-bucket'..key,'last',last + span,'waitcnt',waitCnt)
		tmWait = last + span - current
	else 
		tmWait = span - (current - last) % span
		redis.call('HMSET','2d1b74349305508b-bucket'..key,'last',current + tmWait,'waitcnt',waitCnt)
	end
else
	redis.call('HMSET','2d1b74349305508b-bucket'..key,'last',last + span,'waitcnt',waitCnt)
	tmWait = last + span - current
end
return tmWait
`
	// BucketCheckAddr : 桶限流check脚本，用于校验当前规则缓存的请求数目是否已经超过了max限制，每次get请求前都应该先调用该脚本
	// Input :
	//	key : 规则id
	// Output:
	//	0 : ok，当前缓存的请求数目未超过max限制
	// -1 : 当前缓存的请求数目已超过max限制
	// -2 : 规则不存在
	BucketCheckAddr = `
local key = KEYS[1] --key
local limitInfos = redis.call('HMGET','2d1b74349305508b-bucket'..key,'waitcnt','max')

local waitCnt = tonumber(limitInfos[1])
local max = tonumber(limitInfos[2])

if (waitCnt == nil or max == nil) then	
	return -2
end

local  retValue = 0
if (waitCnt >= max) then
	retValue = -1
else 
	retValue = 0
	waitCnt = waitCnt + 1
	redis.call('HSET','2d1b74349305508b-bucket'..key,'waitcnt',waitCnt)
end

return retValue
`
	// BucketSetAddr : 桶限流set脚本，具体参见BucketAddStr
	BucketSetAddr = BucketAddStr

	// BucketDelAddr : 桶限流del脚本，删除规则对应的所有信息
	// Input :
	//	key : 规则id
	BucketDelAddr = `
local key = KEYS[1] --key
redis.call('HDEL','2d1b74349305508b-bucket'..key,'limit','duration','span','last','max','waitcnt')
`
	// TokenAddStr : token限流脚本，新增一条token限流，token限流以固定速率产生token 并在token溢出(>max)时抛弃产生的token，并每次get请求时减少
	//	相应token数目，当当前token<=0返回错误信息。rate用于表示token的产生速率，calstart表示上次计算token数目的时间点，left表示上次get后剩余token数目
	//	max为当前token桶的最大量
	// Input :
	//	key : 规则id，实际存储key为2d1b74349305508b-token
	//	limit : duration时间段内限制访问limit次
	//	duration : 时间段，与limit共同构成duratio时间段内限流limit次语义
	//	current : 当前时间(μs)
	// 	max : token桶的最大数量
	TokenAddStr = `
local key = KEYS[1] --key
local limit = tonumber(ARGV[1]) --限流大小
local duration = tonumber(ARGV[2]) --时长
local current = tonumber(ARGV[3]) --current timestamp
local max = tonumber(ARGV[4]) --current timestamp
local rate = duration/limit
redis.call('HMSET','2d1b74349305508b-token'..key,'limit',limit,'duration',duration,'rate',rate,'calstart',current,'left',0,'max',max)	
`
	// TokenGetStr : token限流get脚本，用于判断当前请求根据规则是否可以放行。该脚本根据current、calstart、rate计算自上次get以来新产生的token数目
	//	并根据left、max值计算当前剩余token数(min(left+new,max)),最后根据当前token判断本次请求是否可以放行
	// Input :
	//	key : 规则id
	//	current : 当前时间 (μs)
	// Output:
	//	0 : ok，本次请求可以放行
	// -1 : 已无多余token，本次请求不可放行
	// -2 : 相应规则信息不存在
	TokenGetStr = `
local key = KEYS[1] --key
local current = tonumber(ARGV[1]) --current timestamp
local limitInfos = redis.call('HMGET','2d1b74349305508b-token'..key,'calstart','rate','left','max')
	
local calStart = tonumber(limitInfos[1])
local rate = tonumber(limitInfos[2])
local left = tonumber(limitInfos[3])
local max = tonumber(limitInfos[4])

if (calStart == nil or rate == nil or left == nil or max == nil) then
    return -2
end
	
local retValue = 0
local tmPassed = current - calStart > 0 and current - calStart or 0
local curCnt = math.floor((tmPassed)/rate)

if (curCnt + left >= max) then
	left = max
else
	left = left + curCnt
end
	
if (left > 0) then
	left = left - 1
	calStart = current
	retValue = left
else 
	retValue = -1
end

redis.call('HMSET','2d1b74349305508b-token'..key,'calstart',calStart,'left',left)
return retValue
`
	// TokenSetStr : token限流set脚本，具体参见TokenAddStr
	TokenSetStr = TokenAddStr

	// TokenDelStr : token限流删除脚本，删除所有规则相关的信息
	// Input:
	//	key : 规则id
	TokenDelStr = `
local key = KEYS[1] --key
redis.call('HDEL','2d1b74349305508b-token'..key,'limit','duration','rate','calstart','left','max')	
`
)
