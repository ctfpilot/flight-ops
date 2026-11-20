package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ctfpilot/flight-ops/api/v1alpha1"
	jwtV5 "github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testRedisKeyPrefix = "dispatcher:test"

func TestLoadRateLimitConfigFromEnv(t *testing.T) {
	t.Setenv("REDIS_ADDR", "127.0.0.1:6379")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("REDIS_DB", "2")
	t.Setenv("REDIS_KEY_PREFIX", testRedisKeyPrefix)
	t.Setenv("RATE_LIMIT_TIMEOUT_MS", "250")
	t.Setenv("RATE_LIMIT_LIST_MAX", "10")
	t.Setenv("RATE_LIMIT_LIST_WINDOW_SECONDS", "60")
	t.Setenv("RATE_LIMIT_READ_MAX", "5")
	t.Setenv("RATE_LIMIT_READ_WINDOW_SECONDS", "30")
	t.Setenv("RATE_LIMIT_DEPART_MAX", "2")
	t.Setenv("RATE_LIMIT_DEPART_WINDOW_SECONDS", "10")
	t.Setenv("RATE_LIMIT_REFRESH_MAX", "4")
	t.Setenv("RATE_LIMIT_REFRESH_WINDOW_SECONDS", "20")
	t.Setenv("RATE_LIMIT_LAND_MAX", "1")
	t.Setenv("RATE_LIMIT_LAND_WINDOW_SECONDS", "15")

	config, err := LoadRateLimitConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadRateLimitConfigFromEnv() error = %v", err)
	}

	if config.RedisAddr != "127.0.0.1:6379" {
		t.Fatalf("unexpected RedisAddr: %s", config.RedisAddr)
	}
	if config.RedisDB != 2 {
		t.Fatalf("unexpected RedisDB: %d", config.RedisDB)
	}
	if config.RedisKeyPrefix != testRedisKeyPrefix {
		t.Fatalf("unexpected RedisKeyPrefix: %s", config.RedisKeyPrefix)
	}
	if config.RequestTimeout != 250*time.Millisecond {
		t.Fatalf("unexpected RequestTimeout: %s", config.RequestTimeout)
	}
	if policy := config.Policies[JwtActionDepart]; policy.Max != 2 || policy.Window != 10*time.Second {
		t.Fatalf("unexpected depart policy: %+v", policy)
	}
}

func TestLimiterKeyUsesOnlyIdentityFields(t *testing.T) {
	got := limiterKey(JwtActionDepart, "alice", "terminal-1")
	want := "rl:depart:alice:terminal-1"
	if got != want {
		t.Fatalf("limiterKey() = %q, want %q", got, want)
	}
}

func TestRateLimiterAllowsActionsIndependently(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	t.Cleanup(mr.Close)

	runtimeConfig := RateLimitConfig{
		RedisAddr:      mr.Addr(),
		RedisDB:        0,
		RedisKeyPrefix: testRedisKeyPrefix,
		RequestTimeout: time.Second,
		Policies: map[string]LimitPolicy{
			JwtActionList:    {Max: 1, Window: time.Minute},
			JwtActionRead:    {Max: 1, Window: time.Minute},
			JwtActionDepart:  {Max: 1, Window: time.Minute},
			JwtActionLand:    {Max: 1, Window: time.Minute},
			JwtActionRefresh: {Max: 1, Window: time.Minute},
		},
	}

	limiter, err := NewRateLimiter(runtimeConfig)
	if err != nil {
		t.Fatalf("NewRateLimiter() error = %v", err)
	}
	t.Cleanup(func() { _ = limiter.Close() })

	ctx := context.Background()
	first, err := limiter.Allow(ctx, JwtActionList, "alice", "terminal-1")
	if err != nil {
		t.Fatalf("Allow(list) error = %v", err)
	}
	if first.Reached {
		t.Fatalf("Allow(list) unexpectedly reached limit on first request: %+v", first)
	}

	second, err := limiter.Allow(ctx, JwtActionList, "alice", "terminal-1")
	if err != nil {
		t.Fatalf("Allow(list) second error = %v", err)
	}
	if !second.Reached {
		t.Fatalf("Allow(list) second request should have reached the limit: %+v", second)
	}

	readCtx, err := limiter.Allow(ctx, JwtActionRead, "alice", "terminal-1")
	if err != nil {
		t.Fatalf("Allow(read) error = %v", err)
	}
	if readCtx.Reached {
		t.Fatalf("Allow(read) should not share the list bucket: %+v", readCtx)
	}
}

func TestGetFlightsHandlerReturns429AfterLimitExhaustion(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	t.Cleanup(mr.Close)

	runtimeConfig := RateLimitConfig{
		RedisAddr:      mr.Addr(),
		RedisDB:        0,
		RedisKeyPrefix: testRedisKeyPrefix,
		RequestTimeout: time.Second,
		Policies: map[string]LimitPolicy{
			JwtActionList:    {Max: 1, Window: time.Minute},
			JwtActionRead:    {Max: 1, Window: time.Minute},
			JwtActionDepart:  {Max: 1, Window: time.Minute},
			JwtActionLand:    {Max: 1, Window: time.Minute},
			JwtActionRefresh: {Max: 1, Window: time.Minute},
		},
	}

	rlimiter, err := NewRateLimiter(runtimeConfig)
	if err != nil {
		t.Fatalf("NewRateLimiter() error = %v", err)
	}
	t.Cleanup(func() { _ = rlimiter.Close() })

	metrics := NewDispatcherMetrics(prometheus.NewRegistry())
	apiHandler, err := NewApi("default", "1.0.0", &JwtConfig{
		Secret:     defaultSecret,
		Issuers:    []string{defaultIssuer1},
		Audience:   defaultAudience,
		Expiration: 3600,
	}, 5, "server-secret", "flight.local", rlimiter, metrics)
	if err != nil {
		t.Fatalf("NewApi() error = %v", err)
	}

	scheme := runtime.NewScheme()
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	apiHandler.k8sClient = fake.NewClientBuilder().WithScheme(scheme).Build()

	token, err := generateTestToken(defaultSecret, jwtClaim{
		RegisteredClaims: jwtV5.RegisteredClaims{
			Issuer:    defaultIssuer1,
			Audience:  []string{defaultAudience},
			ExpiresAt: jwtV5.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwtV5.NewNumericDate(time.Now()),
			NotBefore: jwtV5.NewNumericDate(time.Now()),
			ID:        defaultJti,
		},
		Flight: jwtFlight{
			Owner:    defaultOwner,
			Terminal: defaultTerminal,
			Action:   JwtActionList,
		},
	})
	if err != nil {
		t.Fatalf("generateTestToken() error = %v", err)
	}

	req1 := httptest.NewRequest(http.MethodGet, "/api/flights", nil)
	req1.Header.Set("Authorization", "Bearer "+token)
	resp1 := httptest.NewRecorder()
	apiHandler.GetFlightsHandler(resp1, req1)
	if resp1.Code != http.StatusOK {
		t.Fatalf("first request status = %d, body = %s", resp1.Code, resp1.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/flights", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	resp2 := httptest.NewRecorder()
	apiHandler.GetFlightsHandler(resp2, req2)
	if resp2.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want %d", resp2.Code, http.StatusTooManyRequests)
	}
	if got := resp2.Header().Get("Retry-After"); got == "" {
		t.Fatalf("Retry-After header was not set")
	}
}
