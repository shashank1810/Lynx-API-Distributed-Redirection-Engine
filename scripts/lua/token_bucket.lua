-- Token Bucket Rate Limiter (Atomic Lua Script)
--
-- Keys:   KEYS[1] = rate limit key (e.g., "rl:<client_ip>")
-- Args:   ARGV[1] = bucket_size (max tokens)
--         ARGV[2] = refill_rate (tokens per second)
--         ARGV[3] = now (current Unix timestamp in microseconds)
--         ARGV[4] = requested (number of tokens to consume, usually 1)
--
-- Returns: {allowed (0 or 1), remaining_tokens}
--
-- The entire operation is atomic within Redis — no race conditions under concurrency.

local key         = KEYS[1]
local bucket_size = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now         = tonumber(ARGV[3])
local requested   = tonumber(ARGV[4])

-- Retrieve current state or initialize.
local bucket = redis.call("HMGET", key, "tokens", "last_refill")
local tokens      = tonumber(bucket[1])
local last_refill = tonumber(bucket[2])

if tokens == nil then
    -- First request: initialize the bucket to full capacity.
    tokens      = bucket_size
    last_refill = now
end

-- Calculate how many tokens to add based on elapsed time.
local elapsed       = math.max(0, now - last_refill)
local tokens_to_add = elapsed * refill_rate / 1000000  -- microsecond precision
tokens              = math.min(bucket_size, tokens + tokens_to_add)

-- Try to consume the requested tokens.
local allowed   = 0
local remaining = tokens

if tokens >= requested then
    tokens    = tokens - requested
    remaining = tokens
    allowed   = 1
end

-- Persist the updated state with a TTL to auto-expire idle buckets.
local ttl = math.ceil(bucket_size / refill_rate) + 1
redis.call("HMSET", key, "tokens", tokens, "last_refill", now)
redis.call("EXPIRE", key, ttl)

return { allowed, math.floor(remaining) }
