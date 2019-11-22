package redis

const (
	// FixAddStr : fix window lua add script, add a hash key with limit、duration、idx、start、wdwcnt props
	//	idx is the window index, start is the init time of the key, wdwcnt is the request cnt within the window
	// Input:
	// 	key: the redis key
	// 	limit: count of limit within the duration
	// 	duration: the time span
	// 	current: current time (μs)
	FixAddStr = `
local key = KEYS[1] --key
local limit = tonumber(ARGV[1]) --限流大小
local duration = tonumber(ARGV[2]) --时长
local current = tonumber(ARGV[3]) --current timestamp
redis.call('HMSET','2d1b74349305508b-fix'..key,'limit',limit,'duration',duration,'idx',0,'start',current,'wdwcnt',0)
`
	// FixGetStr : fix window lua get script, this script check the request cnt within the index window
	// Input:
	//  key: the redis key
	//	current: current time (μs)
	// Output:
	// 0 : ok
	// -1 : requests has exceeded the limit
	// -2 : redis key not exist
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
		
redis.call('HMSET','2d1b74349305508b-fix'..key,'limit',limit,'duration',duration,'idx',curlIdx,'wdwcnt',curWdwCnt)
return retValue
`

	// FixSetStr : fix window lua set script,see FixAddStr
	FixSetStr = FixAddStr

	// FixDelStr : fix window lua del script
	// Input:
	// 		key : the redis key
	FixDelStr = `
local key = KEYS[1] --key
redis.call('HDEL','2d1b74349305508b-fix'..key,'limit','duration','idx','start','wdwcnt')
`
	// SlideAddStr : slide window lua add script
	// Input:
	//	key : redis key
	// 	limit : limit request count between the time span
	// 	duration : the time span
	// 	current : current time (μs)
	SlideAddStr = `
local key = KEYS[1] --key
local limit = tonumber(ARGV[1]) --限流大小
local duration = tonumber(ARGV[2]) --时长
local current = tonumber(ARGV[3]) --current timestamp
redis.call('HMSET','2d1b74349305508b-slide-h'..key,'limit',limit,'duration',duration,'idx',0)
redis.call('ZREMRANGEBYRANK','2d1b74349305508b-slide-z'..key,0,-1)
`
	// SlideSetStr : slide window lua set string, see SlideAddStr
	SlideSetStr = `
local key = KEYS[1] --key
local limit = tonumber(ARGV[1]) --限流大小
local duration = tonumber(ARGV[2]) --时长
local current = tonumber(ARGV[3]) --current timestamp
redis.call('HMSET','2d1b74349305508b-slide-h'..key,'limit',limit,'duration',duration)
redis.call('ZREMRANGEBYRANK','2d1b74349305508b-slide-z'..key,0,-1)
`

	// SlideDelStr : del lua string from slide window
	// Input:
	//	key : redis key
	SlideDelStr = `
local key = KEYS[1] --key
redis.call('HDEL','2d1b74349305508b-slide-h'..key,'limit','duration','idx')
redis.call('ZREMRANGEBYRANK','2d1b74349305508b-slide-z'..key,0,-1)
`
	// SlideGetStr : slide window lua get script, this script maintains a sortedsort with a unix time score and
	//	a hash table with serveral props
	// Input:
	// 	key : redis key
	// 	current : current time (μs)
	// Output:
	//	0 : ok
	//	-1 : request overflow
	//	-2 : key not exist
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

redis.call('HSET','2d1b74349305508b-slide-h'..key,'idx',curIdx + 1)

local start = current - duration
local cnt = redis.call('ZCOUNT','2d1b74349305508b-slide-z'..key, start, current)
local idxs = redis.call('ZRANGEBYSCORE','2d1b74349305508b-slide-z'..key, start,current)

local retValue = 0
if (cnt >= limit) then
	retValue =  -1
else
	redis.call('ZADD','2d1b74349305508b-slide-z'..key, current, curIdx)
	retValue = limit - cnt - 1
end

-- delete old data
local idxs = redis.call('ZRANGEBYSCORE','2d1b74349305508b-slide-z'..key, 0,start)
for i=1,#idxs do
 	redis.call('ZREM','2d1b74349305508b-slide-z'..key, idxs[i])
end
return retValue
`
	// BucketAddStr : bucket lua add script
	// Input :
	//	key : redis key
	//	limit : request count limit between the time span
	//	duration : time span
	//	current : current time (μs)
	//	max : capacity of the bucket
	BucketAddStr = `
local key = KEYS[1] --key
local limit = tonumber(ARGV[1]) --限流大小
local duration = tonumber(ARGV[2]) --时长
local current = tonumber(ARGV[3]) --current timestamp
local max = tonumber(ARGV[4]) --current timestamp
local span = duration/limit
redis.call('HMSET','2d1b74349305508b-bucket'..key,'limit',limit,'duration',duration,'span',span,'last',0,'max',max,'waitcnt',0)
`
	// BucketGetStr : bucket lua get script
	// Input :
	//	key : redis key
	//	current : current time (μs)
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

if (last == 0) then
	waitCnt = waitCnt - 1	
	redis.call('HMSET','2d1b74349305508b-bucket'..key,'last',current,'waitcnt',waitCnt)
	return 0
end

waitCnt = waitCnt - 1
local tmWait = span - (current - last) % span
redis.call('HMSET','2d1b74349305508b-bucket'..key,'last',current,'waitcnt',waitCnt)
return tmWait
`
	// BucketCheckAddr : bucket lua check script
	// Input :
	//	key : redis key
	// Output:
	//	0 : ok
	// -1 : overflow
	// -2 : key not exist
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
	// BucketSetAddr : bucket lua set script
	BucketSetAddr = BucketAddStr

	// BucketDelAddr : bucket lua del script
	// Input :
	//	key : redis key
	BucketDelAddr = `
local key = KEYS[1] --key
redis.call('HDEL','2d1b74349305508b-bucket'..key,'limit','duration','span','last','max','waitcnt')
`
	// TokenAddStr : add lua string for token
	// Input :
	//	key : redis key
	//	limit : request count limit between the time span
	//	duration : time span
	//	current : current time (μs)
	// 	max : capacity of tokens
	TokenAddStr = `
local key = KEYS[1] --key
local limit = tonumber(ARGV[1]) --限流大小
local duration = tonumber(ARGV[2]) --时长
local current = tonumber(ARGV[3]) --current timestamp
local max = tonumber(ARGV[4]) --current timestamp
local rate = duration/limit
redis.call('HMSET','2d1b74349305508b-token'..key,'limit',limit,'duration',duration,'rate',rate,'calstart',current,'left',0,'max',max)	
`
	// TokenGetStr : lua get script for token
	// Input :
	//	key : redis key
	//	current : current time (μs)
	// Output:
	//	0 : ok
	// -1 : overflow
	// -2 : key not exist
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
local curCnt = math.floor((current - calStart)/rate)
	
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
	// TokenSetStr : lua set str for token, see TokenAddStr
	TokenSetStr = TokenAddStr

	// TokenDelStr : del str for token
	// Input:
	//	key : redis key
	TokenDelStr = `
local key = KEYS[1] --key
redis.call('HDEL','2d1b74349305508b-token'..key,'limit','duration','rate','calstart','left','max')	
`
)
