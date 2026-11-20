package api

import (
	"fmt"
	"sync/atomic"

	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Api struct {
	// Namespace for the API
	namespace string
	// Version of the API
	version string
	// MaxConcurrentFlights indicates the maximum number of concurrent flights that can be running at the same time.
	maxConcurrentFlights int
	// ServerSecret used to make random re-producible values for the same dispatcher instance.
	serverSecret string
	// FlightDomain is the domain used for the challenge, which is passed to the flight
	flightDomain string
	// rateLimiter enforces per-action quotas.
	rateLimiter *RateLimiter
	// metrics tracks request and limiter behavior.
	metrics *DispatcherMetrics

	health    atomic.Bool
	k8sClient client.Client
	clientSet *kubernetes.Clientset
	jwtConfig *JwtConfig
}

func NewApi(namespace string, version string, jwtConfig *JwtConfig, maxConcurrentFlights int, serverSecret string, flightDomain string, rateLimiter *RateLimiter, metrics *DispatcherMetrics) (*Api, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace cannot be empty")
	}
	if version == "" {
		return nil, fmt.Errorf("version cannot be empty")
	}
	if jwtConfig == nil {
		return nil, fmt.Errorf("jwtConfig cannot be nil")
	}
	if err := jwtConfig.IsValid(); err != nil {
		return nil, fmt.Errorf("jwtConfig is invalid: %w", err)
	}
	if maxConcurrentFlights <= 0 {
		return nil, fmt.Errorf("maxConcurrentFlights must be greater than 0")
	}
	if serverSecret == "" {
		return nil, fmt.Errorf("serverSecret cannot be empty")
	}
	if flightDomain == "" {
		return nil, fmt.Errorf("flightDomain cannot be empty")
	}
	if rateLimiter == nil {
		return nil, fmt.Errorf("rateLimiter cannot be nil")
	}
	if metrics == nil {
		return nil, fmt.Errorf("metrics cannot be nil")
	}

	return &Api{
		namespace:            namespace,
		version:              version,
		jwtConfig:            jwtConfig,
		maxConcurrentFlights: maxConcurrentFlights,
		serverSecret:         serverSecret,
		flightDomain:         flightDomain,
		rateLimiter:          rateLimiter,
		metrics:              metrics,
	}, nil
}

// Init the API
func (api *Api) Init() error {
	err := api.initClusterClient()
	if err != nil {
		return err
	}

	api.health.Store(true)

	return nil
}

// Check status of the API
func (api *Api) Status() bool {
	err := api.checkAccess()
	return err == nil
}
