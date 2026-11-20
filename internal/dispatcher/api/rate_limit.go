package api

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	limiter "github.com/ulule/limiter/v3"
	redistore "github.com/ulule/limiter/v3/drivers/store/redis"
)

const (
	defaultRedisDB          = 0
	defaultRedisTimeout     = 500 * time.Millisecond
	defaultRedisKeyPrefix   = "dispatcher:rate-limit"
	defaultRateLimitTimeout = 500 * time.Millisecond
	actionList              = JwtActionList
	actionRead              = JwtActionRead
	actionDepart            = JwtActionDepart
	actionLand              = JwtActionLand
	actionRefresh           = JwtActionRefresh
)

var requiredActions = []string{actionList, actionRead, actionDepart, actionLand, actionRefresh}

// LimitPolicy defines the quota for a single action.
type LimitPolicy struct {
	Max    int64
	Window time.Duration
}

// RateLimitConfig holds the runtime rate limiting configuration.
type RateLimitConfig struct {
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
	RedisKeyPrefix string
	RequestTimeout time.Duration
	Policies       map[string]LimitPolicy
}

// LoadRateLimitConfigFromEnv reads the Redis-backed limiter configuration from the process environment.
func LoadRateLimitConfigFromEnv() (RateLimitConfig, error) {
	config := RateLimitConfig{
		RedisAddr:      strings.TrimSpace(os.Getenv("REDIS_ADDR")),
		RedisPassword:  os.Getenv("REDIS_PASSWORD"),
		RedisKeyPrefix: strings.TrimSpace(os.Getenv("REDIS_KEY_PREFIX")),
		RedisDB:        defaultRedisDB,
		RequestTimeout: defaultRateLimitTimeout,
		Policies:       make(map[string]LimitPolicy, len(requiredActions)),
	}

	if config.RedisAddr == "" {
		return RateLimitConfig{}, fmt.Errorf("REDIS_ADDR is required")
	}

	if redisDB := strings.TrimSpace(os.Getenv("REDIS_DB")); redisDB != "" {
		value, err := strconv.Atoi(redisDB)
		if err != nil || value < 0 {
			return RateLimitConfig{}, fmt.Errorf("invalid REDIS_DB value %q", redisDB)
		}
		config.RedisDB = value
	}

	if config.RedisKeyPrefix == "" {
		config.RedisKeyPrefix = defaultRedisKeyPrefix
	}

	for _, action := range requiredActions {
		policy, err := loadPolicyFromEnv(action)
		if err != nil {
			return RateLimitConfig{}, err
		}
		config.Policies[action] = policy
	}

	if timeout := strings.TrimSpace(os.Getenv("RATE_LIMIT_TIMEOUT_MS")); timeout != "" {
		value, err := strconv.Atoi(timeout)
		if err != nil || value <= 0 {
			return RateLimitConfig{}, fmt.Errorf("invalid RATE_LIMIT_TIMEOUT_MS value %q", timeout)
		}
		config.RequestTimeout = time.Duration(value) * time.Millisecond
	}

	return config, nil
}

func loadPolicyFromEnv(action string) (LimitPolicy, error) {
	maxValue := strings.TrimSpace(os.Getenv(strings.ToUpper("RATE_LIMIT_" + action + "_MAX")))
	if maxValue == "" {
		return LimitPolicy{}, fmt.Errorf("RATE_LIMIT_%s_MAX is required", strings.ToUpper(action))
	}
	max, err := strconv.ParseInt(maxValue, 10, 64)
	if err != nil || max <= 0 {
		return LimitPolicy{}, fmt.Errorf("invalid RATE_LIMIT_%s_MAX value %q", strings.ToUpper(action), maxValue)
	}

	windowValue := strings.TrimSpace(os.Getenv(strings.ToUpper("RATE_LIMIT_" + action + "_WINDOW_SECONDS")))
	if windowValue == "" {
		return LimitPolicy{}, fmt.Errorf("RATE_LIMIT_%s_WINDOW_SECONDS is required", strings.ToUpper(action))
	}
	windowSeconds, err := strconv.Atoi(windowValue)
	if err != nil || windowSeconds <= 0 {
		return LimitPolicy{}, fmt.Errorf("invalid RATE_LIMIT_%s_WINDOW_SECONDS value %q", strings.ToUpper(action), windowValue)
	}

	return LimitPolicy{Max: max, Window: time.Duration(windowSeconds) * time.Second}, nil
}

// RateLimiter manages per-action Redis-backed limiters.
type RateLimiter struct {
	client     *redis.Client
	limiters   map[string]*limiter.Limiter
	requestTTL time.Duration
}

// RateLimitExceededError is returned when the quota has been exhausted.
type RateLimitExceededError struct {
	RetryAfter time.Duration
}

func (e *RateLimitExceededError) Error() string {
	return "rate limit exceeded"
}

// NewRateLimiter creates a Redis-backed limiter for every configured action.
func NewRateLimiter(config RateLimitConfig) (*RateLimiter, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	client := redis.NewClient(&redis.Options{
		Addr:         config.RedisAddr,
		Password:     config.RedisPassword,
		DB:           config.RedisDB,
		DialTimeout:  defaultRedisTimeout,
		ReadTimeout:  defaultRedisTimeout,
		WriteTimeout: defaultRedisTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), defaultRedisTimeout)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	store, err := redistore.NewStoreWithOptions(client, limiter.StoreOptions{Prefix: config.RedisKeyPrefix})
	if err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("creating redis store failed: %w", err)
	}

	limiters := make(map[string]*limiter.Limiter, len(config.Policies))
	for action, policy := range config.Policies {
		limiters[action] = limiter.New(store, limiter.Rate{Limit: policy.Max, Period: policy.Window})
	}

	return &RateLimiter{client: client, limiters: limiters, requestTTL: config.RequestTimeout}, nil
}

// Close releases the underlying Redis client.
func (r *RateLimiter) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}

// Healthy verifies the Redis backend is reachable.
func (r *RateLimiter) Healthy() error {
	if r == nil || r.client == nil {
		return fmt.Errorf("rate limiter is not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultRedisTimeout)
	defer cancel()

	if err := r.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}

	return nil
}

// Allow checks whether the given identity may perform the action.
func (r *RateLimiter) Allow(ctx context.Context, action, owner, terminal string) (limiter.Context, error) {
	if r == nil {
		return limiter.Context{}, fmt.Errorf("rate limiter is nil")
	}

	instance, ok := r.limiters[action]
	if !ok {
		return limiter.Context{}, fmt.Errorf("no rate limit policy configured for action %q", action)
	}

	requestCtx, cancel := context.WithTimeout(ctx, r.requestTTL)
	defer cancel()

	ctxValue, err := instance.Get(requestCtx, limiterKey(action, owner, terminal))
	if err != nil {
		return limiter.Context{}, err
	}

	return ctxValue, nil
}

// Check applies the configured rate limit and records the outcome.
func (api *Api) CheckRateLimit(ctx context.Context, token *jwt) (*RateLimitExceededError, error) {
	if api == nil || api.rateLimiter == nil || token == nil || token.Flight == nil {
		return nil, nil
	}

	limitCtx, err := api.rateLimiter.Allow(ctx, token.Flight.Action, token.Flight.Owner, token.Flight.Terminal)
	if err != nil {
		api.metrics.ObserveRedisError("allow")
		api.metrics.ObserveDecision(token.Flight.Action, "error")
		return nil, err
	}

	if limitCtx.Reached {
		api.metrics.ObserveDecision(token.Flight.Action, "deny")
		return &RateLimitExceededError{RetryAfter: retryAfterFromLimiterContext(limitCtx)}, nil
	}

	api.metrics.ObserveDecision(token.Flight.Action, "allow")
	return nil, nil
}

// Validate ensures the configuration is complete and consistent.
func (c RateLimitConfig) Validate() error {
	if strings.TrimSpace(c.RedisAddr) == "" {
		return fmt.Errorf("redis address cannot be empty")
	}
	if strings.TrimSpace(c.RedisKeyPrefix) == "" {
		return fmt.Errorf("redis key prefix cannot be empty")
	}
	if c.RequestTimeout <= 0 {
		return fmt.Errorf("request timeout must be positive")
	}
	for _, action := range requiredActions {
		policy, ok := c.Policies[action]
		if !ok {
			return fmt.Errorf("missing rate limit policy for action %q", action)
		}
		if policy.Max <= 0 {
			return fmt.Errorf("rate limit max for action %q must be positive", action)
		}
		if policy.Window <= 0 {
			return fmt.Errorf("rate limit window for action %q must be positive", action)
		}
	}
	return nil
}

// limiterKey builds the Redis key from JWT identity fields only.
func limiterKey(action, owner, terminal string) string {
	return strings.Join([]string{"rl", action, owner, terminal}, ":")
}

func retryAfterFromLimiterContext(ctx limiter.Context) time.Duration {
	seconds := ctx.Reset - time.Now().Unix()
	seconds = max(seconds, int64(1))
	return time.Duration(seconds) * time.Second
}
