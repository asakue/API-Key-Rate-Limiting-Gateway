-- Sliding Window Counter & Monthly Quota Lua Script
-- KEYS:
-- 1: ratelimit:{key_id}:min:{curr_minute}
-- 2: ratelimit:{key_id}:min:{prev_minute}
-- 3: ratelimit:{key_id}:sec:{curr_sec}
-- 4: quota:{key_id}:{YYYY_MM}

-- ARGV:
-- 1: limit_per_min (e.g. 60)
-- 2: limit_per_sec (e.g. 10)
-- 3: monthly_quota (e.g. 10000)
-- 4: current_second_in_minute (0-59)

local key_curr_min = KEYS[1]
local key_prev_min = KEYS[2]
local key_curr_sec = KEYS[3]
local key_quota    = KEYS[4]

local limit_min   = tonumber(ARGV[1])
local limit_sec   = tonumber(ARGV[2])
local quota_limit = tonumber(ARGV[3])
local sec_offset  = tonumber(ARGV[4]) -- 0-59 seconds

-- 1. Check Monthly Quota
local curr_quota = tonumber(redis.call('GET', key_quota) or '0')
if quota_limit > 0 and curr_quota >= quota_limit then
    return {0, 0, 86400, 0, "MONTHLY_QUOTA_EXCEEDED"}
end

-- 2. Check Per-Second Burst Limit
if limit_sec > 0 then
    local curr_sec_count = tonumber(redis.call('GET', key_curr_sec) or '0')
    if curr_sec_count >= limit_sec then
        local quota_rem = 0
        if quota_limit > 0 then quota_rem = quota_limit - curr_quota end
        return {0, 0, 1, quota_rem, "BURST_LIMIT_EXCEEDED"}
    end
end

-- 3. Check Sliding Window Minute Limit
local curr_min_count = tonumber(redis.call('GET', key_curr_min) or '0')
local prev_min_count = tonumber(redis.call('GET', key_prev_min) or '0')

-- Weight factor: how much of the previous minute overlaps with the sliding window
-- (60 - sec_offset) / 60
local weight = (60.0 - sec_offset) / 60.0
local estimated = math.floor(curr_min_count + (prev_min_count * weight))

local reset_seconds = 60 - sec_offset
if reset_seconds <= 0 then reset_seconds = 60 end

if limit_min > 0 and estimated >= limit_min then
    local quota_rem = 0
    if quota_limit > 0 then quota_rem = quota_limit - curr_quota end
    return {0, 0, reset_seconds, quota_rem, "RATE_LIMIT_EXCEEDED"}
end

-- 4. In-Limit: Increment all counters
redis.call('INCR', key_curr_min)
redis.call('EXPIRE', key_curr_min, 120) -- keep for 2 minutes to serve as prev_min

if limit_sec > 0 then
    redis.call('INCR', key_curr_sec)
    redis.call('EXPIRE', key_curr_sec, 2)
end

if quota_limit > 0 then
    redis.call('INCR', key_quota)
    redis.call('EXPIRE', key_quota, 32 * 86400) -- expire after ~month
end

local remaining_min = 0
if limit_min > 0 then
    remaining_min = limit_min - (estimated + 1)
    if remaining_min < 0 then remaining_min = 0 end
end

local quota_remaining = -1
if quota_limit > 0 then
    quota_remaining = quota_limit - (curr_quota + 1)
end

return {1, remaining_min, reset_seconds, quota_remaining, "OK"}

