package middleware

import (
	"sync"
	"time"
)

// TokenBucket implements a token bucket rate limiter
type TokenBucket struct {
	capacity      int           // Maximum number of tokens
	tokens        int           // Current number of tokens
	refillRate    int           // Tokens added per interval
	refillInterval time.Duration // How often to add tokens
	lastRefill    time.Time     // Last time tokens were added
	mu            sync.Mutex    // Protects the token bucket
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(capacity, refillRate int, refillInterval time.Duration) *TokenBucket {
	return &TokenBucket{
		capacity:       capacity,
		tokens:         capacity, // Start with full bucket
		refillRate:     refillRate,
		refillInterval: refillInterval,
		lastRefill:     time.Now(),
	}
}

// TakeToken attempts to consume a token from the bucket
func (tb *TokenBucket) TakeToken() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	
	tb.refill()
	
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}
	
	return false
}

// TakeTokens attempts to consume multiple tokens from the bucket
func (tb *TokenBucket) TakeTokens(count int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	
	tb.refill()
	
	if tb.tokens >= count {
		tb.tokens -= count
		return true
	}
	
	return false
}

// AvailableTokens returns the current number of available tokens
func (tb *TokenBucket) AvailableTokens() int {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	
	tb.refill()
	return tb.tokens
}

// refill adds tokens to the bucket based on elapsed time
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)
	
	if elapsed >= tb.refillInterval {
		intervals := int(elapsed / tb.refillInterval)
		tokensToAdd := intervals * tb.refillRate
		
		tb.tokens += tokensToAdd
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
		
		tb.lastRefill = now
	}
}

// StreamingFlowController manages flow control for video streaming
type StreamingFlowController struct {
	tokenBucket   *TokenBucket
	connectionLimiter *ConnectionLimiter
	mu            sync.RWMutex
	stats         FlowControlStats
}

// FlowControlStats tracks flow control statistics
type FlowControlStats struct {
	TotalRequests    int64 `json:"total_requests"`
	RateLimited      int64 `json:"rate_limited"`
	ConnectionLimited int64 `json:"connection_limited"`
	Accepted         int64 `json:"accepted"`
}

// NewStreamingFlowController creates a new flow controller
func NewStreamingFlowController(maxConnections, tokensPerSecond int) *StreamingFlowController {
	return &StreamingFlowController{
		tokenBucket:      NewTokenBucket(tokensPerSecond*2, tokensPerSecond, time.Second),
		connectionLimiter: NewConnectionLimiter(maxConnections),
		stats:            FlowControlStats{},
	}
}

// CheckAccess checks if a request can proceed
func (sfc *StreamingFlowController) CheckAccess() (bool, string) {
	sfc.mu.Lock()
	sfc.stats.TotalRequests++
	sfc.mu.Unlock()
	
	// Check rate limiting first (cheaper check)
	if !sfc.tokenBucket.TakeToken() {
		sfc.mu.Lock()
		sfc.stats.RateLimited++
		sfc.mu.Unlock()
		return false, "rate_limited"
	}
	
	// Check connection limiting
	if !sfc.connectionLimiter.Acquire() {
		sfc.mu.Lock()
		sfc.stats.ConnectionLimited++
		sfc.mu.Unlock()
		return false, "connection_limited"
	}
	
	sfc.mu.Lock()
	sfc.stats.Accepted++
	sfc.mu.Unlock()
	
	return true, "accepted"
}

// ReleaseConnection releases a connection slot
func (sfc *StreamingFlowController) ReleaseConnection() {
	sfc.connectionLimiter.Release()
}

// GetStats returns current flow control statistics
func (sfc *StreamingFlowController) GetStats() FlowControlStats {
	sfc.mu.RLock()
	defer sfc.mu.RUnlock()
	
	return FlowControlStats{
		TotalRequests:     sfc.stats.TotalRequests,
		RateLimited:       sfc.stats.RateLimited,
		ConnectionLimited: sfc.stats.ConnectionLimited,
		Accepted:          sfc.stats.Accepted,
	}
}

// GetDetailedStats returns detailed flow control information
func (sfc *StreamingFlowController) GetDetailedStats() map[string]interface{} {
	stats := sfc.GetStats()
	
	return map[string]interface{}{
		"requests": stats,
		"tokens": map[string]interface{}{
			"available": sfc.tokenBucket.AvailableTokens(),
			"capacity":  sfc.tokenBucket.capacity,
		},
		"connections": map[string]interface{}{
			"active":    sfc.connectionLimiter.GetActiveConnections(),
			"max":       sfc.connectionLimiter.GetMaxConnections(),
		},
	}
}