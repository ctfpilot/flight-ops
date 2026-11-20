package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const metricLabelAction = "action"

const unauthenticatedAction = "unauthenticated"

// DispatcherMetrics groups the operational metrics exposed by the dispatcher.
type DispatcherMetrics struct {
	requests    *prometheus.CounterVec
	duration    *prometheus.HistogramVec
	decisions   *prometheus.CounterVec
	redisErrors *prometheus.CounterVec
}

// NewDispatcherMetrics registers the collectors used by the dispatcher.
func NewDispatcherMetrics(reg prometheus.Registerer) *DispatcherMetrics {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	factory := promauto.With(reg)
	return &DispatcherMetrics{
		requests: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "dispatcher_requests_total",
			Help: "Total dispatcher HTTP requests grouped by route, action, and status",
		}, []string{"route", metricLabelAction, "status"}),
		duration: factory.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "dispatcher_request_duration_seconds",
			Help:    "Dispatcher HTTP request latency grouped by route and action",
			Buckets: prometheus.DefBuckets,
		}, []string{"route", metricLabelAction}),
		decisions: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "dispatcher_rate_limit_decisions_total",
			Help: "Total rate-limit decisions grouped by action and outcome",
		}, []string{metricLabelAction, "decision"}),
		redisErrors: factory.NewCounterVec(prometheus.CounterOpts{
			Name: "dispatcher_redis_errors_total",
			Help: "Total Redis errors grouped by operation",
		}, []string{"operation"}),
	}
}

// ObserveRequest records an HTTP request.
func (m *DispatcherMetrics) ObserveRequest(route, action string, status int, duration time.Duration) {
	if m == nil {
		return
	}
	statusLabel := strconv.Itoa(status)
	m.requests.WithLabelValues(route, action, statusLabel).Inc()
	m.duration.WithLabelValues(route, action).Observe(duration.Seconds())
}

// ObserveDecision records a rate-limit outcome.
func (m *DispatcherMetrics) ObserveDecision(action, decision string) {
	if m == nil {
		return
	}
	m.decisions.WithLabelValues(action, decision).Inc()
}

// ObserveRedisError records a Redis failure.
func (m *DispatcherMetrics) ObserveRedisError(operation string) {
	if m == nil {
		return
	}
	m.redisErrors.WithLabelValues(operation).Inc()
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func newStatusRecorder(w http.ResponseWriter) *statusRecorder {
	return &statusRecorder{ResponseWriter: w, status: http.StatusOK}
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Status() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}
