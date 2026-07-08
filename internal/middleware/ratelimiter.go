// Package middleware provides HTTP middleware for cross-cutting concerns.
package middleware

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/raizel/gateway/internal/config"
	"github.com/raizel/gateway/internal/telemetry"
)

// tokenBucketScript is the Lua script for atomic rate limiting.
var tokenBucketScript *redis.Script

func init() {
	luaScript, err := os.ReadFile("scripts/lua/token_bucket.lua")
	if err != nil {
		// Fallback: embed a minimal version if file not found.
		luaScript = []byte(`
local key=KEYS[1] local bs=tonumber(ARGV[1]) local rr=tonumber(ARGV[2])
local now=tonumber(ARGV[3]) local req=tonumber(ARGV[4])
local b=redis.call("HMGET",key,"tokens","last_refill")
local t=tonumber(b[1]) local lr=tonumber(b[2])
if t==nil then t=bs lr=now end
local e=math.max(0,now-lr) t=math.min(bs,t+e*rr/1000000)
local a=0 local r=t
if t>=req then t=t-req r=t a=1 end
redis.call("HMSET",key,"tokens",t,"last_refill",now)
redis.call("EXPIRE",key,math.ceil(bs/rr)+1)
return {a,math.floor(r)}`)
	}
	tokenBucketScript = redis.NewScript(string(luaScript))
}

// RateLimiter returns middleware that enforces per-IP token-bucket rate limiting via Redis + Lua.
func RateLimiter(client *redis.Client, cfg *config.RateLimitConfig, metrics *telemetry.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			clientIP := extractClientIP(r)
			key := cfg.KeyPrefix + clientIP
			now := time.Now().UnixMicro()

			result, err := tokenBucketScript.Run(r.Context(), client, []string{key},
				cfg.BucketSize,
				cfg.Rate,
				now,
				1, // consume 1 token
			).Int64Slice()

			if err != nil {
				// On Redis failure, allow the request (fail-open).
				next.ServeHTTP(w, r)
				return
			}

			allowed := result[0]
			remaining := result[1]

			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", cfg.BucketSize))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

			if allowed == 0 {
				metrics.RateLimitDenied.Inc()
				w.Header().Set("Retry-After", "1")
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}

			metrics.RateLimitAllowed.Inc()
			next.ServeHTTP(w, r)
		})
	}
}

// extractClientIP gets the client IP from X-Forwarded-For, X-Real-IP, or RemoteAddr.
func extractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
