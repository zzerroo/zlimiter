package redis

const (
	AddStr = `
local key = KEYS[1] --key
local limit = tonumber(ARGV[1]) --限流大小
local duration = tonumber(ARGV[2]) --时长
redis.call('HMSET',key,'limit',limit,'duration',duration)
redis.call('SET',"prefix"..key,0,'PX',duration)
`
	// -2 : the key has deleted
	// -1 : reached the limit
	// others : the left times
	GetStr = `
local key = KEYS[1] --限流KEY（一秒一个）
local limitInfos = redis.call('HMGET',key,'limit','duration')
local current = tonumber(redis.call('GET', "prefix"..key) or "-1")
    
local limit = tonumber(limitInfos[1])
local duration = tonumber(limitInfos[2])
    
if (limit == nil or duration == nil) then
    return -2
end
    
if (current + 1 > limit) then --超过限流大小
    return -1
elseif (current == -1) then
	redis.call('SET',"prefix"..key,1,'PX',duration)
	current = 0
else
    redis.call('INCRBY',"prefix"..key,1)
end
    
return limit - current - 1
`

	SetStr = `
local key = KEYS[1] --key
local limit = tonumber(ARGV[1]) --限流大小
local duration = tonumber(ARGV[2]) --时长
redis.call('HMSET',key,'limit',limit,'duration',duration)
redis.call('SET',"prefix"..key,0,'PX',duration)
`

	DelStr = `
local key = KEYS[1]
redis.call('HDEL',key,'limit','duration')
redis.call('DEL',"prefix"..key)
`
)
