package ratelimit

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed script.lua
var rateLimitLuaScript string

type Result struct {
	Allowed        bool
	Remaining      int64
	ResetSeconds   int64
	QuotaRemaining int64
	Reason         string
}

type Limiter struct {
	client *redis.Client
	script *redis.Script
}

func NewLimiter(client *redis.Client) *Limiter {
	script := redis.NewScript(rateLimitLuaScript)
	return &Limiter{
		client: client,
		script: script,
	}
}

// Allow checks whether the request is permitted by rate limit and monthly quota
func (l *Limiter) Allow(ctx context.Context, keyID string, limitPerMin, limitPerSec, monthlyQuota int) (Result, error) {
	if l.client == nil {
		// Redis disabled / unavailable fallback: allow request
		return Result{Allowed: true, Remaining: int64(limitPerMin), ResetSeconds: 60, QuotaRemaining: int64(monthlyQuota), Reason: "FALLBACK_REDIS_NIL"}, nil
	}

	now := time.Now().UTC()
	currMinute := now.Unix() / 60
	prevMinute := currMinute - 1
	currSecond := now.Unix()
	monthKey := now.Format("2006_01")
	secOffset := now.Second()

	keys := []string{
		fmt.Sprintf("ratelimit:%s:min:%d", keyID, currMinute),
		fmt.Sprintf("ratelimit:%s:min:%d", keyID, prevMinute),
		fmt.Sprintf("ratelimit:%s:sec:%d", keyID, currSecond),
		fmt.Sprintf("quota:%s:%s", keyID, monthKey),
	}

	args := []interface{}{
		limitPerMin,
		limitPerSec,
		monthlyQuota,
		secOffset,
	}

	res, err := l.script.Run(ctx, l.client, keys, args...).Slice()
	if err != nil {
		log.Printf("[WARN] Rate limiter script error: %v (fail-open)", err)
		return Result{Allowed: true, Remaining: int64(limitPerMin), ResetSeconds: 60, QuotaRemaining: int64(monthlyQuota), Reason: "FALLBACK_ON_ERROR"}, nil
	}

	if len(res) < 5 {
		return Result{Allowed: true, Remaining: 0, ResetSeconds: 60, QuotaRemaining: 0, Reason: "MALFORMED_SCRIPT_OUTPUT"}, nil
	}

	allowedInt, _ := res[0].(int64)
	remaining, _ := res[1].(int64)
	resetSeconds, _ := res[2].(int64)
	quotaRem, _ := res[3].(int64)
	reason, _ := res[4].(string)

	return Result{
		Allowed:        allowedInt == 1,
		Remaining:      remaining,
		ResetSeconds:   resetSeconds,
		QuotaRemaining: quotaRem,
		Reason:         reason,
	}, nil
}

