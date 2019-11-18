package redis

const (
	// FixAddStr : fix window add lua str
	// key: the redis key
	// limit: count of limit within the duration
	// duration: the time span
	// current: current time (ms)
	FixAddStr = `
local key = KEYS[1] --key
local limit = tonumber(ARGV[1]) --限流大小
local duration = tonumber(ARGV[2]) --时长
local current = tonumber(ARGV[3]) --current timestamp
redis.call('HMSET',key,'limit',limit,'duration',duration,'idx',0,'start',current,'wdwcnt',0)
`
	// FixGetStr : fix window get lua str
	// Input:
	//  key: the redis key
	//	current: current time (ms)
	// Output:
	// 0 : ok
	// -1 : requests has exceeded the limit
	// -2 : redis key not exist
	FixGetStr = `
local key = KEYS[1] --key
local current = tonumber(ARGV[1]) --current timestamp
local limitInfos = redis.call('HMGET',key,'limit','duration','start','idx','wdwcnt')
	
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
		
redis.call('HMSET',key,'limit',limit,'duration',duration,'idx',curlIdx,'wdwcnt',curWdwCnt)
return retValue
`

	// FixSetStr : fix window set lua string,see FixAddStr
	FixSetStr = FixAddStr

	// FixDelStr : del lua string for fix window
	// Input:
	// 		key : the redis key
	FixDelStr = `
local key = KEYS[1] --key
redis.call('HDEL',key,'limit','duration','idx','start','wdwcnt')
`
	// SlideAddStr : slide window add lua string
	// key : redis key
	// limit : limit request count between the time span
	// duration : the time span
	// current : current time (ms)
	SlideAddStr = `
local key = KEYS[1] --key
local limit = tonumber(ARGV[1]) --限流大小
local duration = tonumber(ARGV[2]) --时长
local current = tonumber(ARGV[3]) --current timestamp
redis.call('HMSET',key,'limit',limit,'duration',duration)
`
	// SlideSetStr : set lua string for slide window, see SlideAddStr
	SlideSetStr = `
local key = KEYS[1] --key
local limit = tonumber(ARGV[1]) --限流大小
local duration = tonumber(ARGV[2]) --时长
local current = tonumber(ARGV[3]) --current timestamp
redis.call('HMSET',key,'limit',limit,'duration',duration)
redis.call('ZREMRANGEBYRANK','prefix'..key,0,-1)
`

	// SlideDelStr : del lua string from slide window
	// Input:
	//		key : redis key
	SlideDelStr = `
local key = KEYS[1] --key
redis.call('HDEL',key,'limit','duration')
redis.call('ZREMRANGEBYRANK','prefix'..key,0,-1)
`

	// SlideGetStr : slide window get lua string
	// key : redis key
	// current : current time (ms)
	SlideGetStr = `
local key = KEYS[1] --key
local current = tonumber(ARGV[1]) --current timestamp
local limitInfos = redis.call('HMGET',key,'limit','duration')
	
local limit = tonumber(limitInfos[1])
local duration = tonumber(limitInfos[2])
if (limit == nil or duration == nil ) then
    return -2
end
	
local start = current - duration
local cnt = redis.call('ZCOUNT','prefix'..key, start, current)
	
local retValue = 0
if (cnt >= limit) then
	retValue =  -1
else
	redis.call('ZADD','prefix'..key, current, ARGV[1])
	retValue = limit - cnt - 1
end

print(current,duration,start,cnt)
		
-- delete old data
local idxs = redis.call('ZRANGEBYSCORE','prefix'..key, 0,start)
for i=1,#idxs do
	print(idxs[i])
	redis.call('ZREM','prefix'..key, idxs[i])
end
return retValue
`

	BucketAddStr = `
local key = KEYS[1] --key
local limit = tonumber(ARGV[1]) --限流大小
local duration = tonumber(ARGV[2]) --时长
local current = tonumber(ARGV[3]) --current timestamp
local max = tonumber(ARGV[4]) --current timestamp
local span = duration/limit
redis.call('HMSET',key,'limit',limit,'duration',duration,'span',span,'last',0,'max',max,'waitcnt',0)
`
	BucketGetStr = `
local key = KEYS[1] --key
local current = tonumber(ARGV[1]) --current timestamp
local limitInfos = redis.call('HMGET',key,'span','max','last','waitcnt')
	
local span = tonumber(limitInfos[1])
local max = tonumber(limitInfos[2])
local last = tonumber(limitInfos[3])
local waitCnt = tonumber(limitInfos[4])

if (span == nil or max == nil or last == nil or waitCnt == nil) then
    return -2
end

print(1,key,waitCnt,max)

local retValue = 0
if (waitCnt > max) then
	waitCnt = waitCnt - 1
	redis.call('HMSET',key,'last',current,'waitcnt',waitCnt)
	return -1	
end

if (last == 0) then
	waitCnt = waitCnt - 1	
	redis.call('HMSET',key,'last',current,'waitcnt',waitCnt)
	return 0
end

waitCnt = waitCnt - 1
local tmWait = span - (current - last) % span
redis.call('HMSET',key,'last',current,'waitcnt',waitCnt)
return tmWait
`

	BucketCheckAddr = `
local key = KEYS[1] --key
local limitInfos = redis.call('HMGET',key,'waitcnt','max')

local waitCnt = tonumber(limitInfos[1])
local max = tonumber(limitInfos[2])

print(2,key,waitCnt,max)

local  retValue = 0
if (waitCnt >= max) then
	retValue = -1
else 
	retValue = 0
	waitCnt = waitCnt + 1
	redis.call('HSET',key,'waitcnt',waitCnt)
end

return retValue
`
	BucketSetAddr = BucketAddStr
	BucketDelAddr = `
local key = KEYS[1] --key
redis.call('HDEL',key,'limit','duration','span','last','max','waitcnt')
`
	TokenAddStr = `
local key = KEYS[1] --key
local limit = tonumber(ARGV[1]) --限流大小
local duration = tonumber(ARGV[2]) --时长
local current = tonumber(ARGV[3]) --current timestamp
local max = tonumber(ARGV[4]) --current timestamp
local rate = duration/limit
redis.call('HMSET',key,'limit',limit,'duration',duration,'rate',rate,'calstart',current,'left',0,'max',max)	
`

	TokenGetStr = `
local key = KEYS[1] --key
local current = tonumber(ARGV[1]) --current timestamp
local limitInfos = redis.call('HMGET',key,'calstart','rate','left','max')
	
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
	
redis.call('HMSET',key,'calstart',calStart,'left',left)
return retValue
`

	TokenSetStr = TokenAddStr
	TokenDelStr = `
local key = KEYS[1] --key
redis.call('HDEL',key,'limit','duration','rate','calstart','left','max')	
`
)
