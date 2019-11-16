package redis

const (
	FixAddStr = `
local key = KEYS[1] --key
local limit = tonumber(ARGV[1]) --限流大小
local duration = tonumber(ARGV[2]) --时长
local current = tonumber(ARGV[3]) --current timestamp
redis.call('HMSET',key,'limit',limit,'duration',duration,'idx',0,'start',current,'wdwcnt',0)
`
	// -2 : the key has deleted
	// -1 : reached the limit
	// others : the left times
	FixGetStr = `
local key = KEYS[1] --key
local current = tonumber(ARGV[1]) --current timestamp
local limitInfos = redis.call('HMGET',key,'limit','duration','start','idx','wdwcnt')
	
local limit = tonumber(limitInfos[1])
local duration = tonumber(limitInfos[2])
local start = tonumber(limitInfos[3])
local idx = tonumber(limitInfos[4])
local curWdwCnt = tonumber(limitInfos[5])
	
local retValue = 0
local curlIdx = math.ceil((current - start)/duration)
if (curlIdx ~= idx) then
	retValue = 0
	idx = curlIdx
	curWdwCnt = 1
else
	curWdwCnt = curWdwCnt + 1
	if (curWdwCnt >= limit) then
		retValue = -1
	else
		retValue = 0
	end
end
		
redis.call('HMSET',key,'limit',limit,'duration',duration,'idx',curlIdx,'wdwcnt',curWdwCnt)
return retValue
`

	FixSetStr = `
local key = KEYS[1] --key
local limit = tonumber(ARGV[1]) --限流大小
local duration = tonumber(ARGV[2]) --时长
redis.call('HMSET',key,'limit',limit,'duration',duration)
redis.call('SET',"slide-2d1b74349305508b"..key,0,'PX',duration)
`

	FixDelStr = `
local key = KEYS[1]
redis.call('HDEL',key,'limit','duration')
redis.call('DEL',"slide-2d1b74349305508b"..key)
`

	SlideAddStr = `
local key = KEYS[1] --key
local limit = tonumber(ARGV[1]) --限流大小
local duration = tonumber(ARGV[2]) --时长
local current = tonumber(ARGV[3]) --current timestamp
redis.call('HMSET',key,'limit',limit,'duration',duration)
`
	SlideSetStr = `
`
	SlideDelStr = ``
	SlideGetStr = `
local key = KEYS[1] --key
local current = tonumber(ARGV[1]) --current timestamp
local limitInfos = redis.call('HMGET',key,'limit','duration')
	
local limit = tonumber(limitInfos[1])
local duration = tonumber(limitInfos[2])
	
local start = current - duration
local cnt = redis.call('ZCOUNT','prefix'..key, start, current)
	
local retValue = 0
if (cnt >= limit) then
	retValue =  -1
else
	redis.call('ZADD','prefix'..key, current, ARGV[1])
	retValue = 0
end
		
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
redis.call('HMSET',key,'limit',limit,'duration',duration,'span',span,'last',current,'max',max,'waitcnt',0)
`
	BucketGetStr = `
local key = KEYS[1] --key
local current = tonumber(ARGV[1]) --current timestamp
local limitInfos = redis.call('HMGET',key,'span','max','last','waitcnt')
	
local span = tonumber(limitInfos[1])
local max = tonumber(limitInfos[2])
local last = tonumber(limitInfos[3])
local waitCnt = tonumber(limitInfos[4])
	
waitCnt = waitCnt - 1
	
if (waitCnt >= max) then
	redis.call('HMSET',key,'last',current,'waitcnt',waitCnt)
	return -1
end
	
local tmWait = span - (current - last) % span
redis.call('HMSET',key,'last',current,'waitcnt',waitCnt)
return tmWait
`

	BucketCheckAddr = `
local key = KEYS[1] --key
local limitInfos = redis.call('HMGET',key,'waitcnt','max')

local waitCnt = tonumber(limitInfos[1])
local max = tonumber(limitInfos[2])

local  retValue = 0
if (waitCnt >= max) then
	retValue = -1
else 
	retValue = 0
end

waitCnt = waitCnt + 1

redis.call('HSET',key,'waitcnt',waitCnt)
return retValue
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
	retValue = 0
else 
	retValue = -1
end
	
redis.call('HMSET',key,'calstart',calStart,'left',left)
return retValue
`
)
