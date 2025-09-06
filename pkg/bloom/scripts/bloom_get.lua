local bloomKey = KEYS[1]
local bitsCnt = tonumber(ARGV[1])

-- 添加参数验证
if not bitsCnt or bitsCnt <= 0 then
    return redis.error_reply("Invalid bits count")
end

if not bloomKey or bloomKey == "" then
    return redis.error_reply("Invalid key")
end

for i=1, bitsCnt do
    local offset = tonumber(ARGV[1+i])
    -- 检查offset是否为有效数字且非负
    if not offset or offset < 0 or math.floor(offset) ~= offset then
        return redis.error_reply("Invalid offset: " .. tostring(ARGV[1+i]))
    end
    
    local reply = redis.call('GETBIT', bloomKey, offset)
    if reply == 0 then
        return 0
    end
end

return 1